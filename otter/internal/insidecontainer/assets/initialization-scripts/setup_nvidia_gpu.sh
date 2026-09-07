# mirror_host_ro read-only bind-mounts a host file onto its guest path, reusing
# the host's locked mount flags and resolving symlinks to the backing file.
# Factors out the body shared by the nvidia mount loops below.
# Arguments:
#   src: host path to mount (symlinks are resolved to the backing file)
#   dest: guest path to mount it onto
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   No output on success; performs the read-only bind mount
#   Warning and return 1 if the guest parent dir is read-only
mirror_host_ro()
{
	src="$1"
	dest="$2"

	dest="$(readlink -m "$(dirname "${dest}")")/$(basename "${dest}")"

	dest_parent="$(dirname "${dest}")"
	if [ ! -e "${dest_parent}" ] && ! mkdir -p "${dest_parent}"; then
		printf "Warning: skipping file %s, %s mounted as read-only\n" "${dest}" "${dest_parent}"
		return 1
	fi
	if [ ! -w "${dest_parent}" ]; then
		printf "Warning: skipping file %s, %s mounted as read-only\n" "${dest}" "${dest_parent}"
		return 1
	fi

	if [ -L "${src}" ]; then
		src="$(readlink -fm "${src}")"
	fi

	# Mounting read-only in a user namespace re-checks "locked" flags
	# (nodev,noexec,nosuid); reuse the host's so the bind is permitted.
	locked_flags="$(get_locked_mount_flags "${src}")"
	mount_bind "${src}" "${dest}" ro"${locked_flags:+,${locked_flags}}"
}

# setup_glibc_ldconf points the glibc dynamic linker at the mirrored NVIDIA
# library buckets via /etc/ld.so.conf.d and refreshes the ld.so cache.
# Arguments:
#   None
# Expected global variables:
#   nvidia_libdir: root of the private NVIDIA buckets
# Expected env variables:
#   None
# Outputs:
#   None
setup_glibc_ldconf()
{
	# shellcheck disable=SC2154 # assigned by setup_nvidia_gpu before this is called
	mkdir -p /etc/ld.so.conf.d
	printf '%s\n%s\n' "${nvidia_libdir}/lib64" "${nvidia_libdir}/lib32" \
		> /etc/ld.so.conf.d/00-otter-nvidia.conf
	ldconfig 2> /dev/null
}

# setup_musl_ldpath points the musl dynamic linker at the mirrored NVIDIA
# library buckets by merging the 64-bit bucket into /etc/ld-musl-<arch>.path.
# musl's .path file replaces (rather than appends to) its built-in default of
# /lib:/usr/local/lib:/usr/lib, so an absent file is seeded with those dirs and
# an existing one is preserved untouched; the bucket is only added if missing.
# Arguments:
#   None
# Expected global variables:
#   musl_arch: musl loader arch string (e.g. x86_64), from /lib/ld-musl-*.so.1
#   nvidia_libdir: root of the private NVIDIA buckets
# Expected env variables:
#   None
# Outputs:
#   None
setup_musl_ldpath()
{
	# shellcheck disable=SC2154 # assigned by setup_nvidia_gpu before this is called
	musl_path="/etc/ld-musl-${musl_arch}.path"
	# shellcheck disable=SC2154
	nvidia_bucket="${nvidia_libdir}/lib64"

	if [ ! -e "${musl_path}" ]; then
		printf '/lib:/usr/local/lib:/usr/lib\n%s\n' "${nvidia_bucket}" \
			> "${musl_path}"
	elif ! grep -qxF "${nvidia_bucket}" "${musl_path}"; then
		printf '%s\n' "${nvidia_bucket}" >> "${musl_path}"
	fi
}

# setup_nvidia_gpu integrates the host's nvidia drivers into the guest.
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
setup_nvidia_gpu()
{
printf "otter: Setting up host's nvidia integration...\n"

# Refresh ldconfig cache, also detect if there are empty files remaining
# and clean them.
# This could happen when upgrading drivers and changing versions.
# Use "*.so*" instead of "*.so.*" to also match unversioned .so stubs
# (e.g. libcuda.so, libnvcuvid.so) that block CUDA detection. See #1764.
find /usr/lib* -empty -iname "*.so*" -exec sh -c 'rm -rf "$1" || umount "$1" && rm -rf "$1"' sh {} ';' || :
find /usr/ /etc/ -empty -iname "*nvidia*" -exec sh -c 'rm -rf "$1" || umount "$1" && rm -rf "$1"' sh {} ';' || :

# First we find all generic config files we might need. Skip units under
# systemd/ so host nvidia services aren't mounted over the guest's own.
NVIDIA_FILES="$(find /run/host/etc/ -not -type d \
	-wholename "*nvidia*" \
	-not -wholename "*/systemd/*" || :)"
for nvidia_file in ${NVIDIA_FILES}; do
	mirror_host_ro "${nvidia_file}" "${nvidia_file#/run/host}"
done

# Then we find the non-lib runtime bits the loaders discover by path: EGL
# vendor/external-platform configs, Vulkan and Vulkan SC ICDs and layers,
# OpenCL ICDs, Xorg confs, the OptiX blob and the application-profiles rc the
# GL driver reads for per-app settings. Matched by directory + nvidia name
# rather than exact filenames, so newly shipped files are picked up without
# editing this list.
NVIDIA_CONFS="$(find /run/host/usr/ -not -type d \( \
	-wholename "*/glvnd/egl_vendor.d/*nvidia*" \
	-o -wholename "*/OpenCL/vendors/*nvidia*" \
	-o -wholename "*/X11/xorg.conf.d/*nvidia*" \
	-o -wholename "*/egl_external_platform.d/*nvidia*" \
	-o -wholename "*/nvidia/nvoptix.bin" \
	-o -wholename "*/share/nvidia/*application-profiles*-rc" \
	-o -wholename "*/vulkan/explicit_layer.d/*nvidia*" \
	-o -wholename "*/vulkan/icd.d/*nvidia*" \
	-o -wholename "*/vulkan/implicit_layer.d/*nvidia*" \
	-o -wholename "*/vulkansc/icd.d/*nvidia*" \
	-o -wholename "*nvidia.icd" \
	-o -wholename "*nvidia.yaml" \
	-o -wholename "*nvidia.json" \
	\) || :)"
for nvidia_file in ${NVIDIA_CONFS}; do
	dest_file="${nvidia_file#/run/host}"
	# ICD/vendor JSONs can hard-code an absolute library_path pointing at the
	# host layout. The libraries are relocated into the private bucket, so an
	# absolute path will not resolve in the guest - and the Vulkan loader does
	# not fall back to the basename, so the ICD is silently ignored. Rewrite an
	# absolute library_path to the bare soname, letting ld.so (and thus our
	# ld.so.conf.d bucket) resolve it; everything else is mirrored verbatim.
	case "${nvidia_file}" in
		*.json)
			if grep -q '"library_path"[[:space:]]*:[[:space:]]*"/' "${nvidia_file}" 2> /dev/null; then
				dest_dir="$(dirname "${dest_file}")"
				if mkdir -p "${dest_dir}" 2> /dev/null && [ -w "${dest_dir}" ]; then
					sed 's@\("library_path"[[:space:]]*:[[:space:]]*"\)/[^"]*/\([^"/]*"\)@\1\2@g' \
						"${nvidia_file}" > "${dest_file}"
					continue
				fi
			fi
			;;
		*) ;; # non-JSON configs carry no library_path to rewrite
	esac
	mirror_host_ro "${nvidia_file}" "${dest_file}"
done

# Then we find all the CLI utilities
NVIDIA_BINARIES="$(find /run/host/bin/ /run/host/sbin/ /run/host/usr/bin/ /run/host/usr/sbin/ -not -type d \
	-iname "*nvidia*" || :)"

# Wine/DXVK NVIDIA runtime components. These are Windows PE DLLs, not ELF
# binaries, so they are mirrored 1:1 like the CLI utilities above rather
# than routed through the ELF-class bucketing below.
NVIDIA_WINE="$(find /run/host/usr/lib*/ -not -type d \( \
	-iname "*nvngx*" \
	\) || :)"
for nvidia_file in ${NVIDIA_BINARIES} ${NVIDIA_WINE}; do
	mirror_host_ro "${nvidia_file}" "${nvidia_file#/run/host}"
done

# nVidia userspace libraries.
#
# Linker-resolved libs (libcuda, libGLX_nvidia, libnvidia-*, ...) go in a
# private dir an /etc/ld.so.conf.d entry (below) points the linker at, so
# they never clash with the guest's own libGL/Mesa, split into lib32/lib64
# by ELF class (header byte 5: 1=32, 2=64) so a lib's two builds never collide.
#
# Fixed-path plugins - the Xorg/GBM/VDPAU drivers - are dlopen'd by a
# compiled-in "<libdir>/<plugin>" path, not the linker, so instead they go
# under the guest's native lib root for their ELF class (keeping the
# "<plugin>/..." tail), again so the 32/64 builds never collide.
nvidia_libdir="/usr/lib/otter-nvidia"
mkdir -p "${nvidia_libdir}/lib64" "${nvidia_libdir}/lib32"

# Native library root of the guest for each ELF class. 64-bit is probed
# directly; the 32-bit root is derived from the distro convention
guest_lib64="/usr/lib"
for libroot in /usr/lib/x86_64-linux-gnu /usr/lib64 /usr/lib; do
	[ -d "${libroot}" ] && {
		guest_lib64="${libroot}"
		break
	}
done
case "${guest_lib64}" in
	*/x86_64-linux-gnu) guest_lib32="/usr/lib/i386-linux-gnu" ;; # Debian/Ubuntu multiarch
	*/lib64)
		# Fedora/SUSE keep 32-bit in /usr/lib; on Arch /usr/lib64 is a
		# symlink to the 64-bit /usr/lib, so its 32-bit tree is /usr/lib32.
		if [ "$(readlink -m /usr/lib64)" = "$(readlink -m /usr/lib)" ]; then
			guest_lib32="/usr/lib32"
		else
			guest_lib32="/usr/lib"
		fi
		;;
	*) guest_lib32="/usr/lib32" ;; # bare /usr/lib is 64-bit (Arch-like)
esac
# Single-tree distros: if 32-bit still resolves onto the 64-bit root, skip
# 32-bit plugins rather than shadow it.
[ "$(readlink -m "${guest_lib32}")" = "$(readlink -m "${guest_lib64}")" ] && guest_lib32=""

NVIDIA_LIBS="$(find /run/host/usr/lib*/ -not -type d \( \
	-iname "*lib*nvidia*.so*" \
	-o -iname "*nvidia*.so*" \
	-o -iname "libcuda*.so*" \
	-o -iname "libnvcuvid*" \
	-o -iname "libnvoptix*" \
	\) || :)"
for nvidia_lib in ${NVIDIA_LIBS}; do
	# Resolve symlinks to the real ELF we mount and read the class from.
	real_lib="${nvidia_lib}"
	if [ -L "${real_lib}" ]; then
		real_lib="$(readlink -fm "${real_lib}")"
	fi

	# ELF class of the backing file (byte 5: 1 = 32-bit, 2 = 64-bit) picks
	# both the guest's native root and our private bucket.
	if [ "$(od -An -t u1 -j 4 -N 1 "${real_lib}" 2> /dev/null | tr -d ' ')" = "1" ]; then
		nvidia_class_root="${guest_lib32}"
		nvidia_bucket="${nvidia_libdir}/lib32"
	else
		nvidia_class_root="${guest_lib64}"
		nvidia_bucket="${nvidia_libdir}/lib64"
	fi

	case "${nvidia_lib}" in
		*/xorg/* | */gbm/* | */vdpau/*)
			# Fixed-path plugin: guest's native root for its class,
			# keeping the "<plugin>/..." path below the host library dir.
			[ -n "${nvidia_class_root}" ] || continue
			nvidia_sub="${nvidia_lib#/run/host}"
			case "${nvidia_sub}" in
				/usr/lib/*-linux-*/*) nvidia_sub="${nvidia_sub#/usr/lib/*-linux-*/}" ;;
				/usr/lib64/*) nvidia_sub="${nvidia_sub#/usr/lib64/}" ;;
				/usr/lib32/*) nvidia_sub="${nvidia_sub#/usr/lib32/}" ;;
				/usr/lib/*) nvidia_sub="${nvidia_sub#/usr/lib/}" ;;
				*) ;;
			esac
			dest_file="${nvidia_class_root}/${nvidia_sub}"
			;;
		*)
			# Linker library: isolated dir, bucketed by ELF class.
			dest_file="${nvidia_bucket}/$(basename "${nvidia_lib}")"
			;;
	esac

	# If the destination already exists, there is nothing to do.
	if [ -e "${dest_file}" ]; then
		continue
	fi

	mirror_host_ro "${real_lib}" "${dest_file}"
done

# Point the dynamic linker at the NVIDIA buckets with each libc's own
# mechanism: glibc via /etc/ld.so.conf.d (+ldconfig cache), musl via
# /etc/ld-musl-<arch>.path (see setup_musl_ldpath).
musl_arch=""
for musl_loader in /lib/ld-musl-*.so.1; do
	[ -e "${musl_loader}" ] || continue
	musl_arch="${musl_loader#/lib/ld-musl-}"
	musl_arch="${musl_arch%.so.1}"
	break
done

if [ -n "${musl_arch}" ]; then
	setup_musl_ldpath
else
	setup_glibc_ldconf
fi

}
