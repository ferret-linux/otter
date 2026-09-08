/*
 * bevy-palette.h
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

#include <gdk/gdk.h>

G_BEGIN_DECLS

#define BEVY_TYPE_PALETTE (bevy_palette_get_type())

typedef struct _BevyPaletteScarf
{
  GdkRGBA foreground;
  GdkRGBA background;
} BevyPaletteScarf;

#define BEVY_PALETTE_SCARF_VISUAL_BELL 0
#define BEVY_PALETTE_SCARF_SUPERUSER   1
#define BEVY_PALETTE_SCARF_REMOTE      2
#define BEVY_PALETTE_N_SCARVES         3

typedef struct _BevyPaletteFace
{
  GdkRGBA background;
  GdkRGBA foreground;
  GdkRGBA titlebar_background;
  GdkRGBA titlebar_foreground;
  GdkRGBA cursor_bg;
  GdkRGBA cursor_fg;
  GdkRGBA indexed[16];
  union {
    BevyPaletteScarf scarves[BEVY_PALETTE_N_SCARVES];
    struct {
      BevyPaletteScarf visual_bell;
      BevyPaletteScarf superuser;
      BevyPaletteScarf remote;
    };
  };
} BevyPaletteFace;

G_DECLARE_FINAL_TYPE (BevyPalette, bevy_palette, BEVY, PALETTE, GObject)

GListModel              *bevy_palette_get_all                (void);
void                     bevy_palette_init_user_palettes     (void);
GListModel              *bevy_palette_list_model_get_default (void);
BevyPalette           *bevy_palette_lookup                 (const char     *name);
BevyPalette           *bevy_palette_new_from_file          (const char     *file,
                                                                GError        **error);
BevyPalette           *bevy_palette_new_from_resource      (const char     *file,
                                                                GError        **error);
const char              *bevy_palette_get_id                 (BevyPalette  *self);
const char              *bevy_palette_get_name               (BevyPalette  *self);
const BevyPaletteFace *bevy_palette_get_face               (BevyPalette  *self,
                                                                gboolean        dark);
gboolean                 bevy_palette_use_system_accent      (BevyPalette  *self);
gboolean                 bevy_palette_has_dark               (BevyPalette  *self);
gboolean                 bevy_palette_has_light              (BevyPalette  *self);
char                    *bevy_get_user_palettes_dir          (void);

G_END_DECLS
