/* bevy-shortcut-accel-dialog.h
 *
 * Copyright 2017-2023 Christian Hergert <chergert@redhat.com>
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

G_BEGIN_DECLS

#define BEVY_TYPE_SHORTCUT_ACCEL_DIALOG (bevy_shortcut_accel_dialog_get_type())

G_DECLARE_FINAL_TYPE (BevyShortcutAccelDialog, bevy_shortcut_accel_dialog, BEVY, SHORTCUT_ACCEL_DIALOG, AdwDialog)

GtkWidget  *bevy_shortcut_accel_dialog_new                     (void);
char       *bevy_shortcut_accel_dialog_dup_accelerator         (BevyShortcutAccelDialog *self);
void        bevy_shortcut_accel_dialog_set_accelerator         (BevyShortcutAccelDialog *self,
                                                                  const char                *accelerator);
const char *bevy_shortcut_accel_dialog_get_shortcut_title      (BevyShortcutAccelDialog *self);
void        bevy_shortcut_accel_dialog_set_shortcut_title      (BevyShortcutAccelDialog *self,
                                                                  const char                *title);
const char *bevy_shortcut_accel_dialog_get_default_accelerator (BevyShortcutAccelDialog *self);
void        bevy_shortcut_accel_dialog_set_default_accelerator (BevyShortcutAccelDialog *self,
                                                                  const char                *default_accelerator);

G_END_DECLS

