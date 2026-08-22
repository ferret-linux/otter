# network_check verifies network connectivity before otter-init does
# anything else. Exits the container's entrypoint entirely on failure,
# since otter-init has nothing useful left to do without network access.
#
# Tier 1: reach a handful of well known sites using the best available
# tool (curl, then wget, then ping, in that order). The first successful
# attempt is enough to consider the network up.
#
# Tier 2 (fallback, only reached if every tier 1 attempt failed, or none
# of curl/wget/ping exist at all): verify DNS (getent) and, if curl is
# available, TCP (via curl's own exit code: 6 = DNS failure, 7 = TCP
# connect failure, anything else means we got past both). If wget is the
# only tool available (or none at all), TCP can't be verified
# independently of tier 1, so DNS alone decides the outcome. Passing
# tier 2 still allows startup, but with a warning, since the well known
# sites themselves were unreachable (common on filtered CI/enterprise
# networks that block everything but the traffic they expect).
#
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   Warning on degraded-but-usable network, error on no network
network_check()
{
	network_check_sites="google.com cloudflare.com amazon.com microsoft.com wikipedia.org"
	network_check_timeout=3

	# Pick the first available reachability tool, in order of preference.
	network_check_tool=""
	if command -v curl > /dev/null 2>&1; then
		network_check_tool="curl"
	elif command -v wget > /dev/null 2>&1; then
		network_check_tool="wget"
	elif command -v ping > /dev/null 2>&1; then
		network_check_tool="ping"
	fi

	# Tier 1: try reaching the well known sites with the chosen tool.
	# First success is enough, no need to try the rest.
	if [ -n "${network_check_tool}" ]; then
		for network_check_site in ${network_check_sites}; do
			case "${network_check_tool}" in
				curl)
					if curl --silent --show-error --connect-timeout "${network_check_timeout}" \
						--max-time "${network_check_timeout}" --output /dev/null \
						"https://${network_check_site}" 2> /dev/null; then
						return 0
					fi
					;;
				wget)
					if wget --quiet --spider --timeout="${network_check_timeout}" --tries=1 \
						"https://${network_check_site}" 2> /dev/null; then
						return 0
					fi
					;;
				ping)
					if ping -c 1 -W "${network_check_timeout}" "${network_check_site}" > /dev/null 2>&1; then
						return 0
					fi
					;;
				*)
					;;
			esac
		done
	fi

	# Tier 2 (fallback): none of the well known sites were reachable, or no
	# reachability tool exists at all. Verify DNS and, where possible, TCP
	# independently before giving up entirely.
	network_check_dns_ok=1
	if command -v getent > /dev/null 2>&1; then
		for network_check_site in ${network_check_sites}; do
			if getent hosts "${network_check_site}" > /dev/null 2>&1; then
				network_check_dns_ok=0
				break
			fi
		done
	fi

	# TCP can only be checked independently of tier 1 when curl is
	# available, since its exit code distinguishes a DNS failure (6) from
	# a TCP connect failure (7). Without curl there is no standard,
	# dependency-free way left to test raw TCP in POSIX sh, so TCP is
	# treated as unverifiable rather than guessed at.
	network_check_tcp_ok=1
	network_check_tcp_verifiable=1
	if command -v curl > /dev/null 2>&1; then
		network_check_tcp_verifiable=0
		for network_check_site in ${network_check_sites}; do
			curl --silent --show-error --connect-timeout "${network_check_timeout}" \
				--max-time "${network_check_timeout}" --output /dev/null \
				"https://${network_check_site}" 2> /dev/null
			network_check_rc="$?"
			# Anything other than "couldn't resolve host" (6) or "failed to
			# connect" (7) means we got past the TCP connect stage.
			if [ "${network_check_rc}" -ne 6 ] && [ "${network_check_rc}" -ne 7 ]; then
				network_check_tcp_ok=0
				break
			fi
		done
	fi

	if [ "${network_check_dns_ok}" -eq 0 ] && { [ "${network_check_tcp_verifiable}" -ne 0 ] || [ "${network_check_tcp_ok}" -eq 0 ]; }; then
		printf "Warning: could not reach any well known site, but the network looks otherwise usable, continuing\n"
		return 0
	fi

	printf "Error: no network connectivity, otter-init requires network access\n"
	exit 1
}