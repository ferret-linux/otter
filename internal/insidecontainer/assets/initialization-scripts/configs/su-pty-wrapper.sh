#!/bin/sh

for i do
  [ "${i}" = --pty ] || set -- "$@" "${i}"
  shift
done

# shellcheck disable=SC2117 # forwarding an interactive `su` invocation as-is (minus --pty), not running a fixed command via su
/bin/su "$@"