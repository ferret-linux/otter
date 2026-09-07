/*
 * bevy-client.h
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

#include <gio/gio.h>
#include <vte/vte.h>

#include "bevy-agent-ipc.h"
#include "bevy-profile.h"

G_BEGIN_DECLS

#define BEVY_TYPE_CLIENT (bevy_client_get_type())

G_DECLARE_FINAL_TYPE (BevyClient, bevy_client, BEVY, CLIENT, GObject)

BevyClient       *bevy_client_new                        (gboolean              use_sandbox,
                                                              GError              **error);
const char         *bevy_client_get_user_data_dir          (BevyClient         *self);
void                bevy_client_force_exit                 (BevyClient         *self);
VtePty             *bevy_client_create_pty                 (BevyClient         *self,
                                                              GError              **error);
int                 bevy_client_create_pty_producer        (BevyClient         *self,
                                                              VtePty               *pty,
                                                              GError              **error);
char              **bevy_client_discover_proxy_environment (BevyClient         *self,
                                                              GCancellable         *cancellable,
                                                              GError              **error);
void                bevy_client_discover_shell_async       (BevyClient         *self,
                                                              GCancellable         *cancellable,
                                                              GAsyncReadyCallback   callback,
                                                              gpointer              user_data);
char               *bevy_client_discover_shell_finish      (BevyClient         *client,
                                                              GAsyncResult         *result,
                                                              GError              **error);
void                bevy_client_spawn_async                (BevyClient         *self,
                                                              BevyIpcContainer   *container,
                                                              BevyProfile        *profile,
                                                              const char           *default_shell,
                                                              const char           *last_working_directory_uri,
                                                              VtePty               *pty,
                                                              const char * const   *alt_argv,
                                                              GCancellable         *cancellable,
                                                              GAsyncReadyCallback   callback,
                                                              gpointer              user_data);
BevyIpcProcess   *bevy_client_spawn_finish               (BevyClient         *self,
                                                              GAsyncResult         *result,
                                                              GError              **error);
BevyIpcContainer *bevy_client_discover_current_container (BevyClient         *self,
                                                              VtePty               *pty);
const char         *bevy_client_get_os_name                (BevyClient         *self);
gboolean            bevy_client_ping                       (BevyClient         *self,
                                                              int                   timeout_msec,
                                                              GError              **error);

G_END_DECLS
