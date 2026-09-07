/*
 * bevy-tab.h
 *
 * Copyright 2023-2024 Christian Hergert <chergert@redhat.com>
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

#include <gtk/gtk.h>

#include "bevy-agent-ipc.h"
#include "bevy-profile.h"
#include "bevy-terminal.h"

G_BEGIN_DECLS

typedef enum _BevyProcessLeader
{
  BEVY_PROCESS_LEADER_KIND_UNKNOWN,
  BEVY_PROCESS_LEADER_KIND_SUPERUSER,
  BEVY_PROCESS_LEADER_KIND_REMOTE,
  BEVY_PROCESS_LEADER_KIND_CONTAINER,
} BevyProcessLeaderKind;

typedef enum _BevyZoomLevel
{
  BEVY_ZOOM_LEVEL_MINUS_14 = 1,
  BEVY_ZOOM_LEVEL_MINUS_13,
  BEVY_ZOOM_LEVEL_MINUS_12,
  BEVY_ZOOM_LEVEL_MINUS_11,
  BEVY_ZOOM_LEVEL_MINUS_10,
  BEVY_ZOOM_LEVEL_MINUS_9,
  BEVY_ZOOM_LEVEL_MINUS_8,
  BEVY_ZOOM_LEVEL_MINUS_7,
  BEVY_ZOOM_LEVEL_MINUS_6,
  BEVY_ZOOM_LEVEL_MINUS_5,
  BEVY_ZOOM_LEVEL_MINUS_4,
  BEVY_ZOOM_LEVEL_MINUS_3,
  BEVY_ZOOM_LEVEL_MINUS_2,
  BEVY_ZOOM_LEVEL_MINUS_1,
  BEVY_ZOOM_LEVEL_DEFAULT,
  BEVY_ZOOM_LEVEL_PLUS_1,
  BEVY_ZOOM_LEVEL_PLUS_2,
  BEVY_ZOOM_LEVEL_PLUS_3,
  BEVY_ZOOM_LEVEL_PLUS_4,
  BEVY_ZOOM_LEVEL_PLUS_5,
  BEVY_ZOOM_LEVEL_PLUS_6,
  BEVY_ZOOM_LEVEL_PLUS_7,
  BEVY_ZOOM_LEVEL_PLUS_8,
  BEVY_ZOOM_LEVEL_PLUS_9,
  BEVY_ZOOM_LEVEL_PLUS_10,
  BEVY_ZOOM_LEVEL_PLUS_11,
  BEVY_ZOOM_LEVEL_PLUS_12,
  BEVY_ZOOM_LEVEL_PLUS_13,
  BEVY_ZOOM_LEVEL_PLUS_14,
} BevyZoomLevel;

typedef enum _BevyTabProgress
{
  BEVY_TAB_PROGRESS_INDETERMINATE,
  BEVY_TAB_PROGRESS_ACTIVE,
  BEVY_TAB_PROGRESS_ERROR,
} BevyTabProgress;

#define BEVY_ZOOM_LEVEL_LAST   (BEVY_ZOOM_LEVEL_PLUS_14+1)
#define BEVY_TYPE_TAB          (bevy_tab_get_type())
#define BEVY_TYPE_TAB_PROGRESS (bevy_tab_progress_get_type())

G_DECLARE_FINAL_TYPE (BevyTab, bevy_tab, BEVY, TAB, GtkWidget)

BevyTab          *bevy_tab_new                                (BevyProfile        *profile);
BevyTerminal     *bevy_tab_get_terminal                       (BevyTab            *self);
BevyProfile      *bevy_tab_get_profile                        (BevyTab            *self);
void                bevy_tab_apply_profile                      (BevyTab            *self,
                                                                   BevyProfile        *new_profile);
BevyIpcProcess   *bevy_tab_get_process                        (BevyTab            *self);
const char         *bevy_tab_get_uuid                           (BevyTab            *self);
const char         *bevy_tab_get_command_line                   (BevyTab            *self);
void                bevy_tab_set_command                        (BevyTab            *self,
                                                                   const char * const   *command);
GIcon              *bevy_tab_dup_indicator_icon                 (BevyTab            *self);
char               *bevy_tab_dup_subtitle                       (BevyTab            *self);
char               *bevy_tab_dup_title                          (BevyTab            *self);
gboolean            bevy_tab_get_ignore_osc_title               (BevyTab            *self);
void                bevy_tab_set_ignore_osc_title               (BevyTab            *self,
                                                                   gboolean              ignore_osc_title);
const char         *bevy_tab_get_title_prefix                   (BevyTab            *self);
void                bevy_tab_set_title_prefix                   (BevyTab            *self,
                                                                   const char           *title_prefix);
char               *bevy_tab_dup_current_directory_uri          (BevyTab            *self);
char               *bevy_tab_dup_previous_working_directory_uri (BevyTab            *self);
void                bevy_tab_set_previous_working_directory_uri (BevyTab            *self,
                                                                   const char           *previous_working_directory_uri);
BevyTabProgress   bevy_tab_get_progress                       (BevyTab            *self);
double              bevy_tab_get_progress_fraction              (BevyTab            *self);
BevyZoomLevel     bevy_tab_get_zoom                           (BevyTab            *self);
void                bevy_tab_set_zoom                           (BevyTab            *self,
                                                                   BevyZoomLevel       zoom);
void                bevy_tab_zoom_in                            (BevyTab            *self);
void                bevy_tab_zoom_out                           (BevyTab            *self);
char               *bevy_tab_dup_zoom_label                     (BevyTab            *self);
void                bevy_tab_raise                              (BevyTab            *self);
gboolean            bevy_tab_is_running                         (BevyTab            *self,
                                                                   char                **cmdline);
void                bevy_tab_force_quit                         (BevyTab            *self);
void                bevy_tab_show_banner                        (BevyTab            *self);
void                bevy_tab_set_needs_attention                (BevyTab            *self,
                                                                   gboolean              needs_attention);
BevyIpcContainer *bevy_tab_dup_container                      (BevyTab            *self);
void                bevy_tab_set_container                      (BevyTab            *self,
                                                                   BevyIpcContainer   *container);
gboolean            bevy_tab_has_foreground_process             (BevyTab            *self,
                                                                   GPid                 *pid,
                                                                   char                **cmdline);
const char         *bevy_tab_get_initial_title                  (BevyTab            *self);
void                bevy_tab_set_initial_title                  (BevyTab            *self,
                                                                   const char           *initial_title);
void                bevy_tab_set_initial_working_directory_uri  (BevyTab            *self,
                                                                   const char           *initial_working_directory_uri);
void                bevy_tab_poll_agent_async                   (BevyTab            *self,
                                                                   GCancellable         *cancellable,
                                                                   GAsyncReadyCallback   callback,
                                                                   gpointer              user_data);
gboolean            bevy_tab_poll_agent_finish                  (BevyTab            *self,
                                                                   GAsyncResult         *result,
                                                                   GError              **error);
void                bevy_tab_open_uri                           (BevyTab            *self,
                                                                   const char           *uri);
char               *bevy_tab_query_working_directory_from_agent (BevyTab            *self);
void                bevy_tab_grab_focus                         (BevyTab            *self);

G_END_DECLS
