/* bevy-application.h
 *
 * Copyright 2023 Christian Hergert
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

#include <adwaita.h>

#include "bevy-agent-ipc.h"
#include "bevy-profile.h"
#include "bevy-settings.h"
#include "bevy-shortcuts.h"
#include "bevy-window.h"

G_BEGIN_DECLS

#define BEVY_TYPE_APPLICATION    (bevy_application_get_type())
#define BEVY_APPLICATION_DEFAULT (BEVY_APPLICATION(g_application_get_default()))

G_DECLARE_FINAL_TYPE (BevyApplication, bevy_application, BEVY, APPLICATION, AdwApplication)

gint64              bevy_application_get_default_rlimit_nofile  (void);
BevyApplication  *bevy_application_new                        (const char           *application_id,
                                                                   GApplicationFlags     flags);
const char         *bevy_application_get_user_data_dir          (BevyApplication    *self);
const char         *bevy_application_get_os_name                (BevyApplication    *self);
BevySettings     *bevy_application_get_settings               (BevyApplication    *self);
BevyShortcuts    *bevy_application_get_shortcuts              (BevyApplication    *self);
const char         *bevy_application_get_system_font_name       (BevyApplication    *self);
gboolean            bevy_application_get_overlay_scrollbars     (BevyApplication    *self);
gboolean            bevy_application_control_is_pressed         (BevyApplication    *self);
void                bevy_application_add_profile                (BevyApplication    *self,
                                                                   BevyProfile        *profile);
void                bevy_application_remove_profile             (BevyApplication    *self,
                                                                   BevyProfile        *profile);
BevyProfile      *bevy_application_dup_default_profile        (BevyApplication    *self);
BevyWindow       *bevy_application_get_active_window         (BevyApplication    *self);
void                bevy_application_set_default_profile        (BevyApplication    *self,
                                                                   BevyProfile        *profile);
BevyProfile      *bevy_application_dup_profile                (BevyApplication    *self,
                                                                   const char           *profile_uuid);
GListModel         *bevy_application_list_profiles              (BevyApplication    *self);
GListModel         *bevy_application_list_containers            (BevyApplication    *self);
BevyIpcContainer *bevy_application_lookup_container           (BevyApplication    *self,
                                                                   const char           *container_id);
void                bevy_application_report_error               (BevyApplication    *self,
                                                                   GType                 subsystem,
                                                                   const GError         *error);
VtePty             *bevy_application_create_pty                 (BevyApplication    *self,
                                                                   GError              **error);
void                bevy_application_spawn_async                (BevyApplication    *self,
                                                                   BevyIpcContainer   *container,
                                                                   BevyProfile        *profile,
                                                                   const char           *last_working_directory_uri,
                                                                   VtePty               *pty,
                                                                   const char * const   *argv,
                                                                   GCancellable         *cancellable,
                                                                   GAsyncReadyCallback   callback,
                                                                   gpointer              user_data);
BevyIpcProcess   *bevy_application_spawn_finish               (BevyApplication    *self,
                                                                   GAsyncResult         *result,
                                                                   GError              **error);
void                bevy_application_wait_async                 (BevyApplication    *self,
                                                                   BevyIpcProcess     *process,
                                                                   GCancellable         *cancellable,
                                                                   GAsyncReadyCallback   callback,
                                                                   gpointer              user_data);
int                 bevy_application_wait_finish                (BevyApplication    *self,
                                                                   GAsyncResult         *result,
                                                                   GError              **error);
BevyIpcContainer *bevy_application_discover_current_container (BevyApplication    *self,
                                                                   VtePty               *pty);
BevyIpcContainer *bevy_application_find_container_by_name     (BevyApplication    *self,
                                                                   const char           *runtime,
                                                                   const char           *name);
void                bevy_application_save_session               (BevyApplication    *self);

G_END_DECLS
