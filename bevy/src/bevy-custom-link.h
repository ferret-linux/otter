/*
 * Copyright 2025 Marco Mastropaolo <marco@mastropaolo.com>
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

#include <glib-object.h>
#include <vte/vte.h>

G_BEGIN_DECLS

#define BEVY_TYPE_CUSTOM_LINK (bevy_custom_link_get_type())

G_DECLARE_FINAL_TYPE (BevyCustomLink, bevy_custom_link, BEVY, CUSTOM_LINK, GObject)

BevyCustomLink *bevy_custom_link_new              (void);
BevyCustomLink *bevy_custom_link_new_with_strings (const char       *pattern,
                                                       const char       *target);
void              bevy_custom_link_set_pattern      (BevyCustomLink *self,
                                                       const char       *pattern);
char             *bevy_custom_link_dup_pattern      (BevyCustomLink *self);
char             *bevy_custom_link_dup_target       (BevyCustomLink *self);
void              bevy_custom_link_set_target       (BevyCustomLink *self,
                                                       const char       *target);
VteRegex         *bevy_custom_link_compile          (BevyCustomLink *self);
char             *bevy_custom_link_substitute       (BevyCustomLink *self,
                                                       const char       *subject);

G_END_DECLS
