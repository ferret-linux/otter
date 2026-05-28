#!/usr/bin/env python3
# shellcheck disable=all
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-project/otter
# Copyright (C) 2026 otter contributors
#
# This file is based on distrobox:
#    https://github.com/89luca89/distrobox
# Copyright (C) 2021 distrobox contributors
#
# otter is free software; you can redistribute it and/or modify it
# under the terms of the GNU General Public License version 3
# as published by the Free Software Foundation.
#
# otter is distributed in the hope that it will be useful, but
# WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
# General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with otter; if not, see <http://www.gnu.org/licenses/>.

import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

HOST_MOUNTS_RO_INIT = [
    "/etc/localtime",
    "/run/systemd/journal",
    "/run/systemd/resolve",
    "/run/systemd/seats",
    "/run/systemd/sessions",
    "/run/systemd/users",
    "/var/lib/systemd/coredump",
    "/var/log/journal",
]

UNIT_TARGETS = [
    "/usr/lib/systemd/system/*.mount",
    "/usr/lib/systemd/system/console-getty.service",
    "/usr/lib/systemd/system/getty@.service",
    "/usr/lib/systemd/system/systemd-machine-id-commit.service",
    "/usr/lib/systemd/system/systemd-binfmt.service",
    "/usr/lib/systemd/system/systemd-tmpfiles*",
    "/usr/lib/systemd/system/systemd-udevd.service",
    "/usr/lib/systemd/system/systemd-udev-trigger.service",
    "/usr/lib/systemd/system/systemd-update-utmp*",
    "/usr/lib/systemd/user/pipewire*",
    "/usr/lib/systemd/user/wireplumber*",
    "/usr/lib/systemd/system/suspend.target",
    "/usr/lib/systemd/system/hibernate.target",
    "/usr/lib/systemd/system/hybrid-sleep.target",
    "/usr/lib/systemd/system/systemd-remount-fs.service",
]

USER_INTEGRATION_SCRIPT = """\
#!/bin/sh
sleep 1
ln -sf /run/host/run/user/$(id -ru)/wayland-* /run/user/$(id -ru)/
ln -sf /run/host/run/user/$(id -ru)/pipewire-* /run/user/$(id -ru)/
find /run/host/run/user/$(id -ru)/ -maxdepth 1 -type f -exec sh -c 'grep -qlE COOKIE $0 && ln -sf $0 /run/user/$(id -ru)/$(basename $0)' {} \\;
mkdir -p /run/user/$(id -ru)/app && ln -sf /run/host/run/user/$(id -ru)/app/* /run/user/$(id -ru)/app/
mkdir -p /run/user/$(id -ru)/at-spi && ln -sf /run/host/run/user/$(id -ru)/at-spi/* /run/user/$(id -ru)/at-spi/
mkdir -p /run/user/$(id -ru)/dbus-1 && ln -sf /run/host/run/user/$(id -ru)/dbus-1/* /run/user/$(id -ru)/dbus-1/
mkdir -p /run/user/$(id -ru)/dconf && ln -sf /run/host/run/user/$(id -ru)/dconf/* /run/user/$(id -ru)/dconf/
mkdir -p /run/user/$(id -ru)/gnupg && ln -sf /run/host/run/user/$(id -ru)/gnupg/* /run/user/$(id -ru)/gnupg/
mkdir -p /run/user/$(id -ru)/keyring && ln -sf /run/host/run/user/$(id -ru)/keyring/* /run/user/$(id -ru)/keyring/
mkdir -p /run/user/$(id -ru)/p11-kit && ln -sf /run/host/run/user/$(id -ru)/p11-kit/* /run/user/$(id -ru)/p11-kit/
mkdir -p /run/user/$(id -ru)/pulse && ln -sf /run/host/run/user/$(id -ru)/pulse/* /run/user/$(id -ru)/pulse/
find /run/user/$(id -ru) -maxdepth 2 -xtype l -delete
"""

USER_INTEGRATION_SERVICE = """\
[Unit]
Description=User runtime integration for UID %i
After=user@%i.service
Requires=user-runtime-dir@%i.service

[Service]
User=%i
Type=oneshot
ExecStart=/usr/local/bin/user-integration

Slice=user-%i.slice
"""


def setup_pre_init(container_id):
    print("otter: Setting up init system...")

    for host_mount in HOST_MOUNTS_RO_INIT:
        result = subprocess.run(["findmnt", host_mount], capture_output=True)
        if result.returncode == 0:
            subprocess.run(["umount", host_mount])

    for symlink in [
        "/run/systemd/coredump",
        "/run/systemd/io.system.ManagedOOM",
        "/run/systemd/notify",
        "/run/systemd/private",
    ]:
        Path(symlink).unlink(missing_ok=True)

    if Path("/run/host/etc/localtime").is_file():
        Path("/etc/localtime").unlink(missing_ok=True)
        Path("/etc/localtime").symlink_to("/run/host/etc/localtime")

    if not Path("/dev/console").exists():
        Path("/dev/console").touch()
    Path("/var/console").unlink(missing_ok=True)
    subprocess.run(["mkfifo", "/var/console"])
    subprocess.Popen(["script", "-c", "cat /var/console", "/dev/null"])

    time.sleep(0.5)

    if subprocess.run(["mount", "--bind", "/dev/pts/0", "/dev/console"]).returncode != 0:
        Path("/var/console").unlink(missing_ok=True)
        Path("/var/console").touch()
        subprocess.run(["mount", "--bind", "/var/console", "/dev/console"])

    if Path("/etc/inittab").exists():
        content = Path("/etc/inittab").read_text()
        content = re.sub(r'^(tty\d::)', r'#\1', content, flags=re.MULTILINE)
        Path("/etc/inittab").write_text(content)

    if Path("/etc/rc.conf").exists():
        content = Path("/etc/rc.conf").read_text()
        content = re.sub(r'#rc_env_allow=".*"', 'rc_env_allow="*"', content)
        content = re.sub(r'#rc_crashed_stop=.*', 'rc_crashed_stop=NO', content)
        content = re.sub(r'#rc_crashed_start=.*', 'rc_crashed_start=YES', content)
        content = re.sub(r'#rc_provide=".*"', 'rc_provide="loopback net"', content)
        Path("/etc/rc.conf").write_text(content)

    if Path("/etc/init.d").exists():
        for f in ["hwdrivers", "hwclock", "modules", "modules-load", "modloop"]:
            Path(f"/etc/init.d/{f}").unlink(missing_ok=True)

    if shutil.which("systemctl"):
        unit_targets = list(UNIT_TARGETS)

        result = subprocess.run(
            ["findmnt", "-no", "SOURCE", "/etc/resolv.conf"],
            capture_output=True, text=True
        )
        mount_source = result.stdout.strip()
        if mount_source and container_id not in mount_source:
            unit_targets.append("/usr/lib/systemd/system/systemd-resolved.service")

        for pattern in unit_targets:
            for unit in Path("/").glob(pattern.lstrip("/")):
                subprocess.run(["systemctl", "mask", unit.name], capture_output=True)


def setup_systemd(container_user_name, container_id):
    setup_pre_init(container_id)

    if Path("/usr/lib/systemd/system/user@.service").exists():
        Path("/usr/local/bin/user-integration").write_text(USER_INTEGRATION_SCRIPT)
        Path("/usr/local/bin/user-integration").chmod(0o755)
        Path("/usr/lib/systemd/system/user-integration@.service").write_text(USER_INTEGRATION_SERVICE)

    print("otter: Firing up init system...")

    subprocess.Popen([
        "sh", "-c",
        f"timeout=120 && sleep 1 && while [ \"${{timeout}}\" -gt 0 ]; do "
        f"systemctl is-system-running | grep -E 'running|degraded' && break; "
        f"echo 'waiting for systemd to come up...\\n' && sleep 1 && timeout=$(( timeout -1 )); "
        f"done && "
        f"systemctl start user@{container_user_name}.service && "
        f"systemctl start user-integration@{container_user_name}.service && "
        f"loginctl enable-linger {container_user_name} || : && "
        f"touch /usr/lib/otter/container.ready && "
        f"echo container_setup_done"
    ])

    if Path("/usr/lib/systemd/systemd").exists():
        os.execv("/usr/lib/systemd/systemd", ["/usr/lib/systemd/systemd", "--system", "--log-target=console", "--unit=multi-user.target"])
    elif Path("/lib/systemd/systemd").exists():
        os.execv("/lib/systemd/systemd", ["/lib/systemd/systemd", "--system", "--log-target=console", "--unit=multi-user.target"])


if __name__ == "__main__":
    container_user_name = sys.argv[1]
    container_id = sys.argv[2] if len(sys.argv) > 2 else ""
    setup_systemd(container_user_name, container_id)