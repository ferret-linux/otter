/* bevy-window.h
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

#include "bevy-profile.h"
#include "bevy-tab.h"

G_BEGIN_DECLS

#define BEVY_TYPE_WINDOW (bevy_window_get_type())

G_DECLARE_FINAL_TYPE (BevyWindow, bevy_window, BEVY, WINDOW, AdwApplicationWindow)

BevyWindow  *bevy_window_new                 (void);
BevyWindow  *bevy_window_new_empty           (void);
BevyWindow  *bevy_window_new_for_command     (BevyProfile      *profile,
                                                  const char * const *argv,
                                                  const char         *cwd_uri);
BevyWindow  *bevy_window_new_for_profile     (BevyProfile      *profile);
void           bevy_window_add_tab             (BevyWindow       *self,
                                                  BevyTab          *tab);
void           bevy_window_add_tab_at_end      (BevyWindow       *self,
                                                  BevyTab          *tab);
BevyTab     *bevy_window_add_tab_for_command (BevyWindow       *self,
                                                  BevyProfile      *profile,
                                                  const char * const *argv,
                                                  const char         *cwd_uri);
GListModel    *bevy_window_list_pages          (BevyWindow       *self);
void           bevy_window_append_tab          (BevyWindow       *self,
                                                  BevyTab          *tab);
BevyProfile *bevy_window_get_active_profile  (BevyWindow       *self);
BevyTab     *bevy_window_get_active_tab      (BevyWindow       *self);
void           bevy_window_set_active_tab      (BevyWindow       *self,
                                                  BevyTab          *active_tab);
void           bevy_window_visual_bell         (BevyWindow       *self);
gboolean       bevy_window_focus_tab_by_uuid   (BevyWindow       *self,
                                                  const char         *uuid);
gboolean       bevy_window_is_animating        (BevyWindow       *self);
void           bevy_window_set_tab_pinned      (BevyWindow       *self,
                                                  BevyTab          *tab,
                                                  gboolean            pinned);

G_END_DECLS
