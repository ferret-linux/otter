/* bevy-toolbox-container.c
 *
 * Copyright 2023 Christian Hergert <chergert@redhat.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */


#include "bevy-toolbox-container.h"

struct _BevyToolboxContainer
{
  BevyPodmanContainer parent_instance;
};

G_DEFINE_TYPE (BevyToolboxContainer, bevy_toolbox_container, BEVY_TYPE_PODMAN_CONTAINER)

static void
bevy_toolbox_container_class_init (BevyToolboxContainerClass *klass)
{
}

static void
bevy_toolbox_container_init (BevyToolboxContainer *self)
{
  bevy_ipc_container_set_icon_name (BEVY_IPC_CONTAINER (self), "container-toolbox-symbolic");
  bevy_ipc_container_set_provider (BEVY_IPC_CONTAINER (self), "toolbox");
}
