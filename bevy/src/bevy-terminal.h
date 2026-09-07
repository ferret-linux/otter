/*
 * bevy-terminal.h
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

#include <vte/vte.h>

#include "bevy-palette.h"

G_BEGIN_DECLS

#define BEVY_TYPE_TERMINAL (bevy_terminal_get_type())

G_DECLARE_FINAL_TYPE (BevyTerminal, bevy_terminal, BEVY, TERMINAL, VteTerminal)

BevyPalette *bevy_terminal_get_palette                   (BevyTerminal *self);
void           bevy_terminal_set_palette                   (BevyTerminal *self,
                                                              BevyPalette  *palette);
const char    *bevy_terminal_get_current_container_name    (BevyTerminal *self);
const char    *bevy_terminal_get_current_container_runtime (BevyTerminal *self);
char          *bevy_terminal_dup_current_directory_uri     (BevyTerminal *self);
char          *bevy_terminal_dup_current_file_uri          (BevyTerminal *self);
gboolean       bevy_terminal_can_paste                     (BevyTerminal *self);
void           bevy_terminal_paste                         (BevyTerminal *self);
void           bevy_terminal_reset_for_size                (BevyTerminal *self);
void           bevy_terminal_update_custom_links_list      (BevyTerminal *self,
                                                              GListModel     *custom_links);

G_END_DECLS
