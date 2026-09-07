#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./python-fix.sh
# If no unversioned python3 binary exists, installs a wrapper at
# /usr/bin/python3 that resolves and execs the installed python3.X
# binary at call time.

if command -v python3 >/dev/null 2>&1; then
    exit 0
fi

real_python="$(find /usr/bin -maxdepth 1 -name 'python3.[0-9]*' 2>/dev/null | head -n1)"

if [ -z "${real_python}" ]; then
    echo "ERROR: no python3.X binary found in /usr/bin" >&2
    exit 1
fi

cat > /usr/bin/python3 <<'EOF'
#!/bin/sh
real="$(find /usr/bin -maxdepth 1 -name 'python3.[0-9]*' 2>/dev/null | head -n1)"
if [ -z "${real}" ]; then
    echo "ERROR: no python3.X binary found in /usr/bin" >&2
    exit 1
fi
exec "${real}" "$@"
EOF

chmod +x /usr/bin/python3
