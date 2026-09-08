/*
 * bevy-session-container.c
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


#include "bevy-agent-compat.h"
#include "bevy-agent-util.h"
#include "bevy-process-impl.h"
#include "bevy-session-container.h"

struct _BevySessionContainer
{
  BevyIpcContainerSkeleton parent_instance;
  char **command_prefix;
};

static void container_iface_init (BevyIpcContainerIface *iface);

G_DEFINE_TYPE_WITH_CODE (BevySessionContainer, bevy_session_container, BEVY_IPC_TYPE_CONTAINER_SKELETON,
                         G_IMPLEMENT_INTERFACE (BEVY_IPC_TYPE_CONTAINER, container_iface_init))

static void
bevy_session_container_finalize (GObject *object)
{
  BevySessionContainer *self = (BevySessionContainer *)object;

  g_clear_pointer (&self->command_prefix, g_strfreev);

  G_OBJECT_CLASS (bevy_session_container_parent_class)->finalize (object);
}

static void
bevy_session_container_class_init (BevySessionContainerClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->finalize = bevy_session_container_finalize;
}

static void
bevy_session_container_init (BevySessionContainer *self)
{
  bevy_ipc_container_set_id (BEVY_IPC_CONTAINER (self), "session");
  bevy_ipc_container_set_provider (BEVY_IPC_CONTAINER (self), "session");
}

BevySessionContainer *
bevy_session_container_new (void)
{
  return g_object_new (BEVY_TYPE_SESSION_CONTAINER, NULL);
}

static gboolean
bevy_session_container_handle_spawn (BevyIpcContainer    *container,
                                       GDBusMethodInvocation *invocation,
                                       GUnixFDList           *in_fd_list,
                                       const char            *cwd,
                                       const char * const    *argv,
                                       GVariant              *in_fds,
                                       GVariant              *in_env)
{
  BevySessionContainer *self = (BevySessionContainer *)container;
  g_autoptr(BevyRunContext) run_context = NULL;
  g_autoptr(BevyIpcProcess) process = NULL;
  g_autoptr(GSubprocess) subprocess = NULL;
  g_autoptr(GUnixFDList) out_fd_list = NULL;
  g_autoptr(GError) error = NULL;
  g_auto(GStrv) env = NULL;
  GDBusConnection *connection;
  g_autofree char *object_path = NULL;
  g_autofree char *guid = NULL;

  g_assert (BEVY_IS_SESSION_CONTAINER (self));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));
  g_assert (G_IS_UNIX_FD_LIST (in_fd_list));
  g_assert (cwd != NULL);
  g_assert (argv != NULL);
  g_assert (in_fds != NULL);
  g_assert (in_env != NULL);

  /* Make sure CWD exists within the user session, it might have
   * come from another container that isn't the same or at a path
   * that is not accessible to the user (say from a sudo shell).
   */
  if (cwd[0] == 0 || !g_file_test (cwd, G_FILE_TEST_IS_DIR) ||
      !g_file_test (cwd, G_FILE_TEST_IS_EXECUTABLE))
    cwd = g_get_home_dir ();

  env = g_get_environ ();

  run_context = bevy_run_context_new ();

  /* If we had to run within Flatpak, escape to host */
  bevy_run_context_push_host (run_context);

  /* Place the process inside a new scope similar to what VTE would do. */
  bevy_run_context_push_scope (run_context);

  /* For the default session, we'll just inherit whatever the session gave us
   * as our environment. For other types of containers, that may be different
   * as you likely want to filter some stateful things out.
   */
  if (bevy_agent_is_sandboxed ())
    bevy_run_context_add_minimal_environment (run_context);
  else
    bevy_run_context_set_environ (run_context, (const char * const *)env);

  /* If a command prefix was specified, add that now */
  if (self->command_prefix != NULL)
    bevy_run_context_append_args (run_context, (const char * const *)self->command_prefix);

  /* Use the spawn helper to copy everything that matters after marshaling
   * out of GVariant format. This will be very much the same for other
   * container providers.
   */
  bevy_agent_push_spawn (run_context, in_fd_list, cwd, argv, in_fds, in_env);

  /* Spawn and export our object to the bus. Note that a weak reference is used
   * for the object on the bus so you must keep the object alive to ensure that
   * it is not removed from the bus. The default BevyProcessImpl does that for
   * us by automatically waiting for the child to exit.
   */
  guid = g_dbus_generate_guid ();
  object_path = g_strdup_printf ("/dev/itznoel/Bevy/Process/%s", guid);
  out_fd_list = g_unix_fd_list_new ();
  connection = g_dbus_method_invocation_get_connection (invocation);
  if (!(subprocess = bevy_run_context_spawn (run_context, &error)) ||
      !(process = bevy_process_impl_new (connection, subprocess, object_path, &error)))
    g_dbus_method_invocation_return_gerror (g_steal_pointer (&invocation), error);
  else
    bevy_ipc_container_complete_spawn (container,
                                         g_steal_pointer (&invocation),
                                         out_fd_list,
                                         object_path);

  return TRUE;
}

static gboolean
bevy_session_container_handle_find_program_in_path (BevyIpcContainer    *container,
                                                      GDBusMethodInvocation *invocation,
                                                      const char            *program)
{
  g_autofree char *path = NULL;

  g_assert (BEVY_IS_SESSION_CONTAINER (container));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));

  if ((path = g_find_program_in_path (program)))
    bevy_ipc_container_complete_find_program_in_path (container,
                                                        g_steal_pointer (&invocation),
                                                        path);
  else
    g_dbus_method_invocation_return_error_literal (g_steal_pointer (&invocation),
                                                   G_IO_ERROR,
                                                   G_IO_ERROR_NOT_FOUND,
                                                   "Not Found");

  return TRUE;
}

static gboolean
bevy_session_container_handle_translate_uri (BevyIpcContainer    *container,
                                               GDBusMethodInvocation *invocation,
                                               const char            *uri)
{
  g_assert (BEVY_IS_SESSION_CONTAINER (container));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));

  bevy_ipc_container_complete_translate_uri (container,
                                               g_steal_pointer (&invocation),
                                               uri);

  return TRUE;
}

static void
container_iface_init (BevyIpcContainerIface *iface)
{
  iface->handle_find_program_in_path = bevy_session_container_handle_find_program_in_path;
  iface->handle_spawn = bevy_session_container_handle_spawn;
  iface->handle_translate_uri = bevy_session_container_handle_translate_uri;
}

void
bevy_session_container_set_command_prefix (BevySessionContainer *self,
                                             const char * const     *command_prefix)
{
  char **copy;

  g_return_if_fail (BEVY_IS_SESSION_CONTAINER (self));

  copy = g_strdupv ((char **)command_prefix);
  g_strfreev (self->command_prefix);
  self->command_prefix = copy;
}
