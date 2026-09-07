/* bevy-podman-container.h
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

#pragma once

#include <json-glib/json-glib.h>

#include "bevy-agent-ipc.h"
#include "bevy-run-context.h"

G_BEGIN_DECLS

#define BEVY_TYPE_PODMAN_CONTAINER (bevy_podman_container_get_type())

G_DECLARE_DERIVABLE_TYPE (BevyPodmanContainer, bevy_podman_container, BEVY, PODMAN_CONTAINER, BevyIpcContainerSkeleton)

struct _BevyPodmanContainerClass
{
  BevyIpcContainerSkeletonClass parent_class;

  gboolean (*deserialize)         (BevyPodmanContainer  *self,
                                   JsonObject             *object,
                                   GError                **error);
  void     (*prepare_run_context) (BevyPodmanContainer  *self,
                                   BevyRunContext       *run_context);
};

gboolean    bevy_podman_container_deserialize  (BevyPodmanContainer  *self,
                                                  JsonObject             *object,
                                                  GError                **error);
const char *bevy_podman_container_lookup_label (BevyPodmanContainer  *self,
                                                  const char             *key);

G_END_DECLS
