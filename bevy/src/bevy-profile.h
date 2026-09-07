/*
 * bevy-profile.h
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

#include <pango/pango.h>
#include <vte/vte.h>

#include "bevy-palette.h"
#include "bevy-custom-link.h"

G_BEGIN_DECLS

#define BEVY_PROFILE_KEY_BACKSPACE_BINDING   "backspace-binding"
#define BEVY_PROFILE_KEY_BOLD_IS_BRIGHT      "bold-is-bright"
#define BEVY_PROFILE_KEY_CELL_HEIGHT_SCALE   "cell-height-scale"
#define BEVY_PROFILE_KEY_CELL_WIDTH_SCALE    "cell-width-scale"
#define BEVY_PROFILE_KEY_CJK_AMBIGUOUS_WIDTH "cjk-ambiguous-width"
#define BEVY_PROFILE_KEY_CUSTOM_COMMAND      "custom-command"
#define BEVY_PROFILE_KEY_DEFAULT_CONTAINER   "default-container"
#define BEVY_PROFILE_KEY_DELETE_BINDING      "delete-binding"
#define BEVY_PROFILE_KEY_EXIT_ACTION         "exit-action"
#define BEVY_PROFILE_KEY_LABEL               "label"
#define BEVY_PROFILE_KEY_LIMIT_SCROLLBACK    "limit-scrollback"
#define BEVY_PROFILE_KEY_LOGIN_SHELL         "login-shell"
#define BEVY_PROFILE_KEY_OPACITY             "opacity"
#define BEVY_PROFILE_KEY_PALETTE             "palette"
#define BEVY_PROFILE_KEY_PRESERVE_CONTAINER  "preserve-container"
#define BEVY_PROFILE_KEY_PRESERVE_DIRECTORY  "preserve-directory"
#define BEVY_PROFILE_KEY_SCROLL_ON_KEYSTROKE "scroll-on-keystroke"
#define BEVY_PROFILE_KEY_SCROLL_ON_OUTPUT    "scroll-on-output"
#define BEVY_PROFILE_KEY_SCROLLBACK_LINES    "scrollback-lines"
#define BEVY_PROFILE_KEY_USE_PROXY           "use-proxy"
#define BEVY_PROFILE_KEY_USE_CUSTOM_COMMAND  "use-custom-command"
#define BEVY_PROFILE_KEY_CUSTOM_LINKS        "custom-links"

typedef enum _BevyExitAction
{
  BEVY_EXIT_ACTION_NONE    = 0,
  BEVY_EXIT_ACTION_RESTART = 1,
  BEVY_EXIT_ACTION_CLOSE   = 2,
} BevyExitAction;

typedef enum _BevyPreserveContainer
{
  BEVY_PRESERVE_CONTAINER_NEVER  = 0,
  BEVY_PRESERVE_CONTAINER_ALWAYS = 1,
} BevyPreserveContainer;

typedef enum _BevyPreserveDirectory
{
  BEVY_PRESERVE_DIRECTORY_NEVER  = 0,
  BEVY_PRESERVE_DIRECTORY_SAFE   = 1,
  BEVY_PRESERVE_DIRECTORY_ALWAYS = 2,
} BevyPreserveDirectory;

typedef enum _BevyCjkAmbiguousWidth
{
  BEVY_CJK_AMBIGUOUS_WIDTH_NARROW = 1,
  BEVY_CJK_AMBIGUOUS_WIDTH_WIDE   = 2,
} BevyCjkAmbiguousWidth;

#define BEVY_TYPE_PROFILE (bevy_profile_get_type())

G_DECLARE_FINAL_TYPE (BevyProfile, bevy_profile, BEVY, PROFILE, GObject)

BevyProfile           *bevy_profile_new                      (const char              *uuid);
BevyProfile           *bevy_profile_duplicate                (BevyProfile           *self);
GSettings               *bevy_profile_dup_settings             (BevyProfile           *self);
const char              *bevy_profile_get_uuid                 (BevyProfile           *self);
char                    *bevy_profile_dup_default_container    (BevyProfile           *self);
void                     bevy_profile_set_default_container    (BevyProfile           *self,
                                                                  const char              *default_container);
char                    *bevy_profile_dup_label                (BevyProfile           *self);
void                     bevy_profile_set_label                (BevyProfile           *self,
                                                                  const char              *label);
gboolean                 bevy_profile_get_limit_scrollback     (BevyProfile           *self);
void                     bevy_profile_set_limit_scrollback     (BevyProfile           *self,
                                                                  gboolean                 limit_scrollback);
int                      bevy_profile_get_scrollback_lines     (BevyProfile           *self);
void                     bevy_profile_set_scrollback_lines     (BevyProfile           *self,
                                                                  int                      scrollback_lines);
gboolean                 bevy_profile_get_scroll_on_keystroke  (BevyProfile           *self);
void                     bevy_profile_set_scroll_on_keystroke  (BevyProfile           *self,
                                                                  gboolean                 scroll_on_keystroke);
gboolean                 bevy_profile_get_scroll_on_output     (BevyProfile           *self);
void                     bevy_profile_set_scroll_on_output     (BevyProfile           *self,
                                                                  gboolean                 scroll_on_output);
gboolean                 bevy_profile_get_bold_is_bright       (BevyProfile           *self);
void                     bevy_profile_set_bold_is_bright       (BevyProfile           *self,
                                                                  gboolean                 bold_is_bright);
double                   bevy_profile_get_cell_height_scale    (BevyProfile           *self);
void                     bevy_profile_set_cell_height_scale    (BevyProfile           *self,
                                                                  double                   cell_height_scale);
double                   bevy_profile_get_cell_width_scale     (BevyProfile           *self);
void                     bevy_profile_set_cell_width_scale     (BevyProfile           *self,
                                                                  double                   cell_width_scale);
BevyExitAction         bevy_profile_get_exit_action          (BevyProfile           *self);
void                     bevy_profile_set_exit_action          (BevyProfile           *self,
                                                                  BevyExitAction         exit_action);
BevyPreserveContainer  bevy_profile_get_preserve_container   (BevyProfile           *self);
void                     bevy_profile_set_preserve_container   (BevyProfile           *self,
                                                                  BevyPreserveContainer  preserve_container);
BevyPreserveDirectory  bevy_profile_get_preserve_directory   (BevyProfile           *self);
void                     bevy_profile_set_preserve_directory   (BevyProfile           *self,
                                                                  BevyPreserveDirectory  preserve_directory);
char                    *bevy_profile_dup_palette_id           (BevyProfile           *self);
BevyPalette           *bevy_profile_dup_palette              (BevyProfile           *self);
void                     bevy_profile_set_palette              (BevyProfile           *self,
                                                                  BevyPalette           *palette);
double                   bevy_profile_get_opacity              (BevyProfile           *self);
void                     bevy_profile_set_opacity              (BevyProfile           *self,
                                                                  double                   opacity);
VteEraseBinding          bevy_profile_get_backspace_binding    (BevyProfile           *self);
void                     bevy_profile_set_backspace_binding    (BevyProfile           *self,
                                                                  VteEraseBinding          backspace_binding);
VteEraseBinding          bevy_profile_get_delete_binding       (BevyProfile           *self);
void                     bevy_profile_set_delete_binding       (BevyProfile           *self,
                                                                  VteEraseBinding          delete_binding);
BevyCjkAmbiguousWidth  bevy_profile_get_cjk_ambiguous_width  (BevyProfile           *self);
void                     bevy_profile_set_cjk_ambiguous_width  (BevyProfile           *self,
                                                                  BevyCjkAmbiguousWidth  cjk_ambiguous_width);
gboolean                 bevy_profile_get_login_shell          (BevyProfile           *self);
void                     bevy_profile_set_login_shell          (BevyProfile           *self,
                                                                  gboolean                 login_shell);
char                    *bevy_profile_dup_custom_command       (BevyProfile           *self);
void                     bevy_profile_set_custom_command       (BevyProfile           *self,
                                                                  const char              *custom_command);
gboolean                 bevy_profile_get_use_custom_command   (BevyProfile           *self);
void                     bevy_profile_set_use_custom_command   (BevyProfile           *self,
                                                                  gboolean                 use_custom_command);
gboolean                 bevy_profile_get_use_proxy            (BevyProfile           *self);
void                     bevy_profile_set_use_proxy            (BevyProfile           *self,
                                                                  gboolean                 use_proxy);
GListModel              *bevy_profile_list_custom_links        (BevyProfile           *self);
void                     bevy_profile_add_custom_link          (BevyProfile           *self,
                                                                  BevyCustomLink        *custom_link);
void                     bevy_profile_undo_remove_custom_link  (BevyProfile           *self,
                                                                  BevyCustomLink        *custom_link,
                                                                  guint                    index);
gboolean                 bevy_profile_remove_custom_link       (BevyProfile           *self,
                                                                  BevyCustomLink        *custom_link,
                                                                  guint                   *index);
void                     bevy_profile_save_custom_link_changes (BevyProfile           *self);

G_END_DECLS
