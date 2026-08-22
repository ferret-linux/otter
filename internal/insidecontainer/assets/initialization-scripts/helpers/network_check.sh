# network_check verifies network connectivity before otter-init does
# anything else. Exits the container's entrypoint entirely on failure,
# since otter-init has nothing useful left to do without network access.
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
network_check()
{
	if ! getent hosts one.one.one.one > /dev/null 2>&1; then
		printf "Error: no network connectivity, otter-init requires network access\n"
		exit 1
	fi
}