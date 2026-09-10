/*
 * bevy-window-dressing.h
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

#include "bevy-palette.h"
#include "bevy-window.h"

G_BEGIN_DECLS

#define BEVY_TYPE_WINDOW_DRESSING (bevy_window_dressing_get_type())

G_DECLARE_FINAL_TYPE (BevyWindowDressing, bevy_window_dressing, BEVY, WINDOW_DRESSING, GObject)

BevyWindowDressing *bevy_window_dressing_new           (BevyWindow         *window);
BevyWindowDressing *bevy_window_dressing_new_for_root  (GtkWidget         *root,
                                                        gboolean            main_contents);
GtkWidget          *bevy_window_dressing_dup_window    (BevyWindowDressing *self);
BevyPalette        *bevy_window_dressing_get_palette   (BevyWindowDressing *self);
void                bevy_window_dressing_set_palette   (BevyWindowDressing *self,
                                                        BevyPalette        *palette);
double              bevy_window_dressing_get_opacity   (BevyWindowDressing *self);
void                bevy_window_dressing_set_opacity   (BevyWindowDressing *self,
                                                        double              opacity);

G_END_DECLS

