/* bevy-fullscreen-box.h
 *
 * Copyright © 2021 Purism SPC
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

G_BEGIN_DECLS

#define BEVY_TYPE_FULLSCREEN_BOX (bevy_fullscreen_box_get_type())

G_DECLARE_FINAL_TYPE (BevyFullscreenBox, bevy_fullscreen_box, BEVY, FULLSCREEN_BOX, GtkWidget)

BevyFullscreenBox *bevy_fullscreen_box_new            (void);
gboolean             bevy_fullscreen_box_get_fullscreen (BevyFullscreenBox *self);
void                 bevy_fullscreen_box_set_fullscreen (BevyFullscreenBox *self,
                                                           gboolean             fullscreen);
gboolean             bevy_fullscreen_box_get_autohide   (BevyFullscreenBox *self);
void                 bevy_fullscreen_box_set_autohide   (BevyFullscreenBox *self,
                                                           gboolean             autohide);
GtkWidget           *bevy_fullscreen_box_get_content    (BevyFullscreenBox *self);
void                 bevy_fullscreen_box_set_content    (BevyFullscreenBox *self,
                                                           GtkWidget           *content);
void                 bevy_fullscreen_box_add_top_bar    (BevyFullscreenBox *self,
                                                           GtkWidget           *child);
void                 bevy_fullscreen_box_add_bottom_bar (BevyFullscreenBox *self,
                                                           GtkWidget           *child);
void                 bevy_fullscreen_box_reveal         (BevyFullscreenBox *self);
void                 bevy_fullscreen_box_unreveal       (BevyFullscreenBox *self);

G_END_DECLS
