# network_check verifies network connectivity before otter-init does
# anything else. Exits the container's entrypoint entirely on failure,
# since otter-init has nothing useful left to do without network access.
#
# Tier 1: for each of a handful of well known sites, try every reachability
# tool that exists on the system (curl, then wget, then ping). The first
# successful attempt, with any tool against any site, is enough to consider
# the network up. Trying every available tool (rather than committing to
# just one) avoids a false failure when one tool happens to be broken or
# misconfigured (e.g. curl missing CA certs) while another would work fine.
#
# Tier 2 (fallback, only reached if every tier 1 attempt failed, or none of
# curl/wget/ping exist at all): verify DNS and TCP independently, each
# using every available standard tool, with a more generous timeout since
# we are already in the degraded path:
#   - DNS: getent, then nslookup, then host, then python3, whichever exist.
#   - TCP: curl's own exit code against the well known sites (6 = DNS
#     failure, 7 = TCP connect failure, anything else means we got past
#     both); if that is not conclusive and nc exists, a raw TCP connect to
#     port 53 on a few well known DNS resolver IPs (a DNS-independent
#     check, since it bypasses hostname resolution entirely); if neither is
#     available, wget as a last, lower-confidence resort.
# TCP is only ever a decision factor when at least one of those tools
# actually exists; if none do, TCP is left unverifiable and the outcome
# rests on DNS alone.
#
# Passing tier 2 still allows startup, but with a warning, since the well
# known sites themselves were unreachable (common on filtered CI/enterprise
# networks that allow real traffic but block or don't recognize ICMP/the
# specific sites tried).
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
	network_check_dns_ips="1.1.1.1 8.8.8.8 9.9.9.9"
	network_check_timeout_t1=3
	network_check_timeout_t2=5

	network_check_have_curl=1
	command -v curl > /dev/null 2>&1 || network_check_have_curl=0
	network_check_have_wget=1
	command -v wget > /dev/null 2>&1 || network_check_have_wget=0
	network_check_have_ping=1
	command -v ping > /dev/null 2>&1 || network_check_have_ping=0
	network_check_have_nc=1
	command -v nc > /dev/null 2>&1 || network_check_have_nc=0
	network_check_have_getent=1
	command -v getent > /dev/null 2>&1 || network_check_have_getent=0
	network_check_have_nslookup=1
	command -v nslookup > /dev/null 2>&1 || network_check_have_nslookup=0
	network_check_have_host=1
	command -v host > /dev/null 2>&1 || network_check_have_host=0
	network_check_have_python3=1
	command -v python3 > /dev/null 2>&1 || network_check_have_python3=0

	# Tier 1: cross-check every available tool against every well known
	# site, first success anywhere is enough.
	for network_check_site in ${network_check_sites}; do
		if [ "${network_check_have_curl}" -eq 1 ]; then
			if curl --silent --show-error --connect-timeout "${network_check_timeout_t1}" \
				--max-time "${network_check_timeout_t1}" --output /dev/null \
				"https://${network_check_site}" 2> /dev/null; then
				return 0
			fi
		fi
		if [ "${network_check_have_wget}" -eq 1 ]; then
			if wget --quiet --spider --timeout="${network_check_timeout_t1}" --tries=1 \
				"https://${network_check_site}" 2> /dev/null; then
				return 0
			fi
		fi
		if [ "${network_check_have_ping}" -eq 1 ]; then
			if ping -c 1 -W "${network_check_timeout_t1}" "${network_check_site}" > /dev/null 2>&1; then
				return 0
			fi
		fi
	done

	# Tier 2 (fallback): none of the well known sites were reachable with
	# any available tool. Verify DNS and TCP independently before failing.

	# 2a: DNS resolution, tried with every available tool.
	network_check_dns_ok=1
	for network_check_site in ${network_check_sites}; do
		if [ "${network_check_have_getent}" -eq 1 ] && getent hosts "${network_check_site}" > /dev/null 2>&1; then
			network_check_dns_ok=0
			break
		fi
		if [ "${network_check_have_nslookup}" -eq 1 ] && nslookup "${network_check_site}" > /dev/null 2>&1; then
			network_check_dns_ok=0
			break
		fi
		if [ "${network_check_have_host}" -eq 1 ] && host "${network_check_site}" > /dev/null 2>&1; then
			network_check_dns_ok=0
			break
		fi
		if [ "${network_check_have_python3}" -eq 1 ] && \
			python3 -c "import socket,sys; sys.exit(0 if socket.gethostbyname('${network_check_site}') else 1)" > /dev/null 2>&1; then
			network_check_dns_ok=0
			break
		fi
	done

	# 2b: TCP reachability, tried with every available tool, most reliable
	# signal first. Only decided by tools that actually exist.
	network_check_tcp_ok=1
	network_check_tcp_verifiable=0

	if [ "${network_check_tcp_ok}" -ne 0 ] && [ "${network_check_have_curl}" -eq 1 ]; then
		network_check_tcp_verifiable=1
		for network_check_site in ${network_check_sites}; do
			curl --silent --show-error --connect-timeout "${network_check_timeout_t2}" \
				--max-time "${network_check_timeout_t2}" --output /dev/null \
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

	if [ "${network_check_tcp_ok}" -ne 0 ] && [ "${network_check_have_nc}" -eq 1 ]; then
		network_check_tcp_verifiable=1
		# Connect straight to a few well known DNS resolver IPs on port 53,
		# bypassing hostname resolution entirely, so this still works even
		# if DNS itself is what's broken.
		for network_check_ip in ${network_check_dns_ips}; do
			if nc -z -w "${network_check_timeout_t2}" "${network_check_ip}" 53 > /dev/null 2>&1; then
				network_check_tcp_ok=0
				break
			fi
		done
	fi

	if [ "${network_check_tcp_ok}" -ne 0 ] && [ "${network_check_have_wget}" -eq 1 ]; then
		network_check_tcp_verifiable=1
		for network_check_site in ${network_check_sites}; do
			if wget --quiet --spider --timeout="${network_check_timeout_t2}" --tries=1 \
				"https://${network_check_site}" 2> /dev/null; then
				network_check_tcp_ok=0
				break
			fi
		done
	fi

	if [ "${network_check_dns_ok}" -eq 0 ] && { [ "${network_check_tcp_verifiable}" -eq 0 ] || [ "${network_check_tcp_ok}" -eq 0 ]; }; then
		printf "Warning: could not reach any well known site, but the network looks otherwise usable, continuing\n"
		return 0
	fi

	printf "Error: no network connectivity, otter-init requires network access\n"
	exit 1
}