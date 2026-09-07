/* bevy-unix-fd-map.h
 *
 * Copyright 2022-2023 Christian Hergert <chergert@redhat.com>
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

#include <gio/gio.h>

G_BEGIN_DECLS

#define BEVY_TYPE_UNIX_FD_MAP (bevy_unix_fd_map_get_type())

G_DECLARE_FINAL_TYPE (BevyUnixFDMap, bevy_unix_fd_map, BEVY, UNIX_FD_MAP, GObject)

BevyUnixFDMap *bevy_unix_fd_map_new             (void);
guint            bevy_unix_fd_map_get_length      (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_peek_stdin      (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_peek_stdout     (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_peek_stderr     (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_steal_stdin     (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_steal_stdout    (BevyUnixFDMap  *self);
int              bevy_unix_fd_map_steal_stderr    (BevyUnixFDMap  *self);
gboolean         bevy_unix_fd_map_steal_from      (BevyUnixFDMap  *self,
                                                     BevyUnixFDMap  *other,
                                                     GError          **error);
int              bevy_unix_fd_map_peek            (BevyUnixFDMap  *self,
                                                     guint             index,
                                                     int              *dest_fd);
int              bevy_unix_fd_map_get             (BevyUnixFDMap  *self,
                                                     guint             index,
                                                     int              *dest_fd,
                                                     GError          **error);
int              bevy_unix_fd_map_steal           (BevyUnixFDMap  *self,
                                                     guint             index,
                                                     int              *dest_fd);
void             bevy_unix_fd_map_take            (BevyUnixFDMap  *self,
                                                     int               source_fd,
                                                     int               dest_fd);
gboolean         bevy_unix_fd_map_open_file       (BevyUnixFDMap  *self,
                                                     const char       *filename,
                                                     int               mode,
                                                     int               dest_fd,
                                                     GError          **error);
int              bevy_unix_fd_map_get_max_dest_fd (BevyUnixFDMap  *self);
gboolean         bevy_unix_fd_map_stdin_isatty    (BevyUnixFDMap  *self);
gboolean         bevy_unix_fd_map_stdout_isatty   (BevyUnixFDMap  *self);
gboolean         bevy_unix_fd_map_stderr_isatty   (BevyUnixFDMap  *self);
GIOStream       *bevy_unix_fd_map_create_stream   (BevyUnixFDMap  *self,
                                                     int               dest_read_fd,
                                                     int               dest_write_fd,
                                                     GError          **error);
gboolean         bevy_unix_fd_map_silence_fd      (BevyUnixFDMap  *self,
                                                     int               dest_fd,
                                                     GError          **error);

G_END_DECLS
