/*
 * bevy-process-impl.c
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


#include <errno.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#include <glib/gstdio.h>

#include "bevy-agent-compat.h"
#include "bevy-agent-impl.h"
#include "bevy-process-impl.h"

struct _BevyProcessImpl
{
  BevyIpcProcessSkeleton parent_instance;
  GSubprocess *subprocess;
  GPid pid;
};

static void process_iface_init (BevyIpcProcessIface *iface);

G_DEFINE_TYPE_WITH_CODE (BevyProcessImpl, bevy_process_impl, BEVY_IPC_TYPE_PROCESS_SKELETON,
                         G_IMPLEMENT_INTERFACE (BEVY_IPC_TYPE_PROCESS, process_iface_init))

static GHashTable *exec_to_kind;

static void
bevy_process_impl_finalize (GObject *object)
{
  BevyProcessImpl *self = (BevyProcessImpl *)object;

  g_clear_object (&self->subprocess);

  G_OBJECT_CLASS (bevy_process_impl_parent_class)->finalize (object);
}

static void
bevy_process_impl_class_init (BevyProcessImplClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->finalize = bevy_process_impl_finalize;

  exec_to_kind = g_hash_table_new (g_str_hash, g_str_equal);
#define ADD_MAPPING(name, kind) \
    g_hash_table_insert (exec_to_kind, (char *)name, (char *)kind)
  ADD_MAPPING ("docker", "container");
  ADD_MAPPING ("flatpak", "container");
  ADD_MAPPING ("mosh", "remote");
  ADD_MAPPING ("mosh-client", "remote");
  ADD_MAPPING ("podman", "container");
  ADD_MAPPING ("rlogin", "remote");
  ADD_MAPPING ("scp", "remote");
  ADD_MAPPING ("sftp", "remote");
  ADD_MAPPING ("slogin", "remote");
  ADD_MAPPING ("ssh", "remote");
  ADD_MAPPING ("telnet", "remote");
  ADD_MAPPING ("toolbox", "container");
#undef ADD_MAPPING
}

static void
bevy_process_impl_init (BevyProcessImpl *self)
{
}

static void
bevy_process_impl_wait_cb (GObject      *object,
                             GAsyncResult *result,
                             gpointer      user_data)
{
  GSubprocess *subprocess = (GSubprocess *)object;
  g_autoptr(BevyProcessImpl) self = user_data;
  g_autoptr(GError) error = NULL;

  g_assert (G_IS_SUBPROCESS (subprocess));
  g_assert (G_IS_ASYNC_RESULT (result));

  g_subprocess_wait_finish (subprocess, result, &error);

  bevy_ipc_agent_emit_process_exited (BEVY_IPC_AGENT (bevy_agent_impl_get_default ()),
                                        g_dbus_interface_skeleton_get_object_path (G_DBUS_INTERFACE_SKELETON (self)),
                                        g_subprocess_get_status (subprocess));

  if (g_subprocess_get_if_signaled (subprocess))
    bevy_ipc_process_emit_signaled (BEVY_IPC_PROCESS (self),
                                      g_subprocess_get_term_sig (subprocess));
  else
    bevy_ipc_process_emit_exited (BEVY_IPC_PROCESS (self),
                                    g_subprocess_get_exit_status (subprocess));

  g_dbus_interface_skeleton_unexport (G_DBUS_INTERFACE_SKELETON (self));

  g_clear_object (&self->subprocess);
}

BevyIpcProcess *
bevy_process_impl_new (GDBusConnection  *connection,
                         GSubprocess      *subprocess,
                         const char       *object_path,
                         GError          **error)
{
  g_autoptr(BevyProcessImpl) self = NULL;
  const char *ident;

  g_return_val_if_fail (G_IS_DBUS_CONNECTION (connection), NULL);
  g_return_val_if_fail (G_IS_SUBPROCESS (subprocess), NULL);
  g_return_val_if_fail (object_path != NULL, NULL);

  ident = g_subprocess_get_identifier (subprocess);

  self = g_object_new (BEVY_TYPE_PROCESS_IMPL, NULL);
  self->pid = atoi (ident);
  g_set_object (&self->subprocess, subprocess);

  g_subprocess_wait_async (subprocess,
                           NULL,
                           bevy_process_impl_wait_cb,
                           g_object_ref (self));

  if (!g_dbus_interface_skeleton_export (G_DBUS_INTERFACE_SKELETON (self),
                                         connection,
                                         object_path,
                                         error))
    return NULL;

  return BEVY_IPC_PROCESS (g_steal_pointer (&self));
}

static gboolean
bevy_process_impl_handle_send_signal (BevyIpcProcess      *process,
                                        GDBusMethodInvocation *invocation,
                                        int                    signum)
{
  BevyProcessImpl *self = (BevyProcessImpl *)process;

  g_assert (BEVY_IS_PROCESS_IMPL (process));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));

  if (self->subprocess != NULL)
    g_subprocess_send_signal (self->subprocess, signum);

  bevy_ipc_process_complete_send_signal (process, g_steal_pointer (&invocation));

  return TRUE;
}

static char *
get_cmdline_for_pid (GPid pid)
{
  char *cmdline = NULL;

#ifdef __linux__
  g_autofree char *path = g_strdup_printf ("/proc/%u/cmdline", (guint)pid);
  _g_autofd int fd = -1;
  ssize_t len;

  if ((fd = open (path, O_RDONLY | O_CLOEXEC)) != -1)
    {
      cmdline = g_malloc (1025);

      do
        len = read (fd, cmdline, 1024);
      while (len == -1 && errno == EINTR);

      if (len > 0)
        {
          cmdline[len] = 0;

          for (ssize_t i = 0; i < len; i++)
            {
              if (cmdline[i] == 0 || g_ascii_iscntrl (cmdline[i]))
                cmdline[i] = ' ';
            }

          if (!g_utf8_validate (cmdline, len, NULL))
            {
              char *valid = g_utf8_make_valid (cmdline, len);

              g_free (cmdline);
              cmdline = valid;
            }
        }
      else
        {
          g_free (cmdline);
          cmdline = NULL;
        }
    }
#endif

  return cmdline;
}

static const char *
get_leader_kind (GPid pid)
{
  g_autofree char *path = NULL;
  g_autofree char *exe = NULL;
  g_autoptr(GFile) file = NULL;
  g_autoptr(GFileInfo) info = NULL;
  const char *leader_kind = NULL;
  char execpath[512];
  gssize len;

  if (pid <= 0)
    return "unknown";

  path = g_strdup_printf ("/proc/%d/", pid);
  exe = g_strdup_printf ("/proc/%d/exe", pid);
  file = g_file_new_for_path (path);

  /* We use GFile API so that we can avoid linking against
   * stat64 which is different on older glibc versions.
   */
  info = g_file_query_info (file, G_FILE_ATTRIBUTE_UNIX_UID, G_FILE_QUERY_INFO_NONE, NULL, NULL);
  if (info && g_file_info_get_attribute_uint32 (info, G_FILE_ATTRIBUTE_UNIX_UID) == 0)
    return "superuser";

  len = readlink (exe, execpath, sizeof execpath-1);

  if (len > 0)
    {
      const char *end;

      execpath[len] = 0;

      if ((end = strrchr (execpath, '/')))
        leader_kind = g_hash_table_lookup (exec_to_kind, end+1);
    }

  if (leader_kind == NULL)
    leader_kind = "unknown";

  return leader_kind;
}

static gboolean
bevy_process_impl_handle_has_foreground_process (BevyIpcProcess      *process,
                                                   GDBusMethodInvocation *invocation,
                                                   GUnixFDList           *in_fd_list,
                                                   GVariant              *in_pty_fd)
{
  BevyProcessImpl *self = (BevyProcessImpl *)process;
  gboolean has_foreground_process = FALSE;
  g_autofree char *cmdline = NULL;
  _g_autofd int pty_fd = -1;
  int pty_fd_handle;
  GPid pid = -1;

  g_assert (BEVY_IS_PROCESS_IMPL (self));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));

  pty_fd_handle = g_variant_get_handle (in_pty_fd);

  if (in_fd_list != NULL)
    pty_fd = g_unix_fd_list_get (in_fd_list, pty_fd_handle, NULL);

  if (pty_fd != -1)
    {
      pid = tcgetpgrp (pty_fd);
      has_foreground_process = pid != self->pid;

      if (pid > 0)
        cmdline = get_cmdline_for_pid (pid);
    }

  bevy_ipc_process_complete_has_foreground_process (process,
                                                      g_steal_pointer (&invocation),
                                                      NULL,
                                                      has_foreground_process,
                                                      pid,
                                                      cmdline ? cmdline : "",
                                                      get_leader_kind (pid));

  return TRUE;
}

static gboolean
bevy_process_impl_handle_get_working_directory (BevyIpcProcess      *process,
                                                  GDBusMethodInvocation *invocation,
                                                  GUnixFDList           *in_fd_list,
                                                  GVariant              *in_pty_fd)
{
  BevyProcessImpl *self = (BevyProcessImpl *)process;
  _g_autofd int pty_fd = -1;
  int pty_fd_handle;
  GPid pid;

  g_assert (BEVY_IS_PROCESS_IMPL (self));
  g_assert (G_IS_DBUS_METHOD_INVOCATION (invocation));

  pty_fd_handle = g_variant_get_handle (in_pty_fd);

  if (in_fd_list != NULL)
    pty_fd = g_unix_fd_list_get (in_fd_list, pty_fd_handle, NULL);

  if (pty_fd != -1)
    pid = tcgetpgrp (pty_fd);
  else
    pid = self->pid;

  if (pid > 0)
    {
      /* TODO: This might need alternatives for non-Linux based kernels,
       *       if that matters at all in places where it can be used.
       */
      g_autofree char *proc_path = g_strdup_printf ("/proc/%d/cwd", pid);
      g_autofree char *cwd = g_file_read_link (proc_path, NULL);

      if (cwd != NULL)
        {
          bevy_ipc_process_complete_get_working_directory (process,
                                                             g_steal_pointer (&invocation),
                                                             NULL,
                                                             cwd);
          return TRUE;
        }
    }

  bevy_ipc_process_complete_get_working_directory (process,
                                                     g_steal_pointer (&invocation),
                                                     NULL,
                                                     "/");

  return TRUE;
}

static void
process_iface_init (BevyIpcProcessIface *iface)
{
  iface->handle_send_signal = bevy_process_impl_handle_send_signal;
  iface->handle_has_foreground_process = bevy_process_impl_handle_has_foreground_process;
  iface->handle_get_working_directory = bevy_process_impl_handle_get_working_directory;
}
