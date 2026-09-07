/* bevy-run-context.h
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

#include "bevy-unix-fd-map.h"

G_BEGIN_DECLS

#define BEVY_TYPE_RUN_CONTEXT (bevy_run_context_get_type())

G_DECLARE_FINAL_TYPE (BevyRunContext, bevy_run_context, BEVY, RUN_CONTEXT, GObject)

/**
 * BevyRunContextShell:
 * @BEVY_RUN_CONTEXT_SHELL_DEFAULT: A basic shell with no user scripts
 * @BEVY_RUN_CONTEXT_SHELL_LOGIN: A user login shell similar to `bash -l`
 * @BEVY_RUN_CONTEXT_SHELL_INTERACTIVE: A user interactive shell similar to `bash -i`
 *
 * Describes the type of shell to be used within the context.
 */
typedef enum _BevyRunContextShell
{
  BEVY_RUN_CONTEXT_SHELL_DEFAULT     = 0,
  BEVY_RUN_CONTEXT_SHELL_LOGIN       = 1,
  BEVY_RUN_CONTEXT_SHELL_INTERACTIVE = 2,
} BevyRunContextShell;

/**
 * BevyRunContextHandler:
 *
 * Returns: %TRUE if successful; otherwise %FALSE and @error must be set.
 */
typedef gboolean (*BevyRunContextHandler) (BevyRunContext    *run_context,
                                             const char * const  *argv,
                                             const char * const  *env,
                                             const char          *cwd,
                                             BevyUnixFDMap     *unix_fd_map,
                                             gpointer             user_data,
                                             GError             **error);

BevyRunContext    *bevy_run_context_new                     (void);
void                 bevy_run_context_push                    (BevyRunContext         *self,
                                                                 BevyRunContextHandler   handler,
                                                                 gpointer                  handler_data,
                                                                 GDestroyNotify            handler_data_destroy);
void                 bevy_run_context_push_host               (BevyRunContext         *self);
void                 bevy_run_context_push_at_base            (BevyRunContext         *self,
                                                                 BevyRunContextHandler   handler,
                                                                 gpointer                  handler_data,
                                                                 GDestroyNotify            handler_data_destroy);
void                 bevy_run_context_push_error              (BevyRunContext         *self,
                                                                 GError                   *error);
void                 bevy_run_context_push_scope              (BevyRunContext         *self);
void                 bevy_run_context_push_shell              (BevyRunContext         *self,
                                                                 BevyRunContextShell     shell);
const char * const  *bevy_run_context_get_argv                (BevyRunContext         *self);
void                 bevy_run_context_set_argv                (BevyRunContext         *self,
                                                                 const char * const       *argv);
const char * const  *bevy_run_context_get_environ             (BevyRunContext         *self);
void                 bevy_run_context_set_environ             (BevyRunContext         *self,
                                                                 const char * const       *environ);
void                 bevy_run_context_add_environ             (BevyRunContext         *self,
                                                                 const char * const       *environ);
void                 bevy_run_context_add_minimal_environment (BevyRunContext         *self);
void                 bevy_run_context_environ_to_argv         (BevyRunContext         *self);
const char          *bevy_run_context_get_cwd                 (BevyRunContext         *self);
void                 bevy_run_context_set_cwd                 (BevyRunContext         *self,
                                                                 const char               *cwd);
void                 bevy_run_context_take_fd                 (BevyRunContext         *self,
                                                                 int                       source_fd,
                                                                 int                       dest_fd);
gboolean             bevy_run_context_merge_unix_fd_map       (BevyRunContext         *self,
                                                                 BevyUnixFDMap          *unix_fd_map,
                                                                 GError                  **error);
void                 bevy_run_context_prepend_argv            (BevyRunContext         *self,
                                                                 const char               *arg);
void                 bevy_run_context_prepend_args            (BevyRunContext         *self,
                                                                 const char * const       *args);
void                 bevy_run_context_append_argv             (BevyRunContext         *self,
                                                                 const char               *arg);
void                 bevy_run_context_append_args             (BevyRunContext         *self,
                                                                 const char * const       *args);
gboolean             bevy_run_context_append_args_parsed      (BevyRunContext         *self,
                                                                 const char               *args,
                                                                 GError                  **error);
void                 bevy_run_context_append_formatted        (BevyRunContext         *self,
                                                                 const char               *format,
                                                                 ...) G_GNUC_PRINTF (2, 3);
const char          *bevy_run_context_getenv                  (BevyRunContext         *self,
                                                                 const char               *key);
void                 bevy_run_context_setenv                  (BevyRunContext         *self,
                                                                 const char               *key,
                                                                 const char               *value);
void                 bevy_run_context_unsetenv                (BevyRunContext         *self,
                                                                 const char               *key);
GIOStream           *bevy_run_context_create_stdio_stream     (BevyRunContext         *self,
                                                                 GError                  **error);
GSubprocessLauncher *bevy_run_context_end                     (BevyRunContext         *self,
                                                                 GError                  **error);
GSubprocess         *bevy_run_context_spawn                   (BevyRunContext         *self,
                                                                 GError                  **error);
GSubprocess         *bevy_run_context_spawn_with_flags        (BevyRunContext         *self,
                                                                 GSubprocessFlags          flags,
                                                                 GError                  **error);

G_END_DECLS
