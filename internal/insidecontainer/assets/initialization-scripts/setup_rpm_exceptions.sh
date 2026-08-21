# setup_rpm_exceptions will create path-excludes for host mounts, rpm only (dnf, zypper).
# Arguments:
#   None
# Expected global variables:
#   init: if this is an initful container
#   HOST_MOUNTS_RO: list of readonly mountpoints, to avoid
#   HOST_MOUNTS: list of readwrite mountpoints, to avoid
# Expected env variables:
#   None
# Outputs:
#   None
setup_rpm_exceptions()
{
	# In case of an RPM distro, we can specify that our bind_mount directories
	# are in fact net shares. This prevents conflicts during package installations.
	if [ "${init}" -eq 0 ]; then
		mkdir -p /usr/lib/rpm/macros.d/
		# Loop through all the environment vars
		# and export them to the container.
		net_mounts=""
		for net_mount in \
			${HOST_MOUNTS_RO} ${HOST_MOUNTS} \
			'/dev' '/proc' '/sys' '/tmp' \
			'/etc/hosts' '/etc/resolv.conf' '/etc/passwd' '/etc/shadow'; do

			net_mounts="${net_mount}:${net_mounts}"
		done
		net_mounts=${net_mounts%?}
		cat << EOF > /usr/lib/rpm/macros.d/macros.otter
%_netsharedpath ${net_mounts}
EOF
	fi
}
