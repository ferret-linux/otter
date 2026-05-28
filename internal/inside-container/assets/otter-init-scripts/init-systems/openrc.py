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
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))
from systemd import setup_pre_init  # type: ignore


def setup_openrc(container_id):
    setup_pre_init(container_id)

    print("otter: Firing up init system...")

    Path("/usr/lib/otter/container.ready").touch()
    print("container_setup_done")

    os.execv("/sbin/init", ["/sbin/init"])


if __name__ == "__main__":
    container_id = sys.argv[1] if len(sys.argv) > 1 else ""
    setup_openrc(container_id)