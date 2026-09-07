/* bevy-run-context.c
 *
 * Copyright 2022 Christian Hergert <chergert@redhat.com>
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

#include "config.h"

#include <errno.h>
#include <string.h>
#include <sys/ioctl.h>
#ifdef __linux__
# include <sys/prctl.h>
#endif
#include <sys/resource.h>
#include <unistd.h>

#include "bevy-agent-compat.h"
#include "bevy-agent-util.h"
#include "bevy-run-context.h"

typedef struct
{
  GList                    qlink;
  char                    *cwd;
  GArray                  *argv;
  GArray                  *env;
  BevyUnixFDMap         *unix_fd_map;
  BevyRunContextHandler  handler;
  gpointer                 handler_data;
  GDestroyNotify           handler_data_destroy;
} BevyRunContextLayer;

struct _BevyRunContext
{
  GObject               parent_instance;
  GQueue                layers;
  BevyRunContextLayer root;
  guint                 ended : 1;
  guint                 setup_tty : 1;
};

G_DEFINE_TYPE (BevyRunContext, bevy_run_context, G_TYPE_OBJECT)

BevyRunContext *
bevy_run_context_new (void)
{
  return g_object_new (BEVY_TYPE_RUN_CONTEXT, NULL);
}

static void
copy_envvar_with_fallback (BevyRunContext   *run_context,
                           const char * const *environ,
                           const char         *key,
                           const char         *fallback)
{
  const char *val;

  if ((val = g_environ_getenv ((char **)environ, key)))
    bevy_run_context_setenv (run_context, key, val);
  else if (fallback != NULL)
    bevy_run_context_setenv (run_context, key, fallback);
}

/**
 * bevy_run_context_add_minimal_environment:
 * @self: a #BevyRunContext
 *
 * Adds a minimal set of environment variables.
 *
 * This is useful to get access to things like the display or other
 * expected variables.
 */
void
bevy_run_context_add_minimal_environment (BevyRunContext *self)
{
  g_auto(GStrv) host_environ = g_get_environ ();
  static const char *copy_env[] = {
    "AT_SPI_BUS_ADDRESS",
    "COLUMNS",
    "DBUS_SESSION_BUS_ADDRESS",
    "DBUS_SYSTEM_BUS_ADDRESS",
    "DESKTOP_SESSION",
    "DISPLAY",
    "HOME",
    "LANG",
    "LINES",
    "SHELL",
    "SSH_AUTH_SOCK",
    "USER",
    "VTE_VERSION",
    "WAYLAND_DISPLAY",
    "XAUTHORITY",
    "XDG_CURRENT_DESKTOP",
    "XDG_DATA_DIRS",
    "XDG_MENU_PREFIX",
    "XDG_RUNTIME_DIR",
    "XDG_SEAT",
    "XDG_SESSION_DESKTOP",
    "XDG_SESSION_ID",
    "XDG_SESSION_TYPE",
    "XDG_VTNR",
  };
  const char *val;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  for (guint i = 0; i < G_N_ELEMENTS (copy_env); i++)
    {
      const char *key = copy_env[i];

      if ((val = g_environ_getenv (host_environ, key)))
        bevy_run_context_setenv (self, key, val);
    }

  copy_envvar_with_fallback (self, (const char * const *)host_environ, "TERM", "xterm-256color");
  copy_envvar_with_fallback (self, (const char * const *)host_environ, "COLORTERM", "truecolor");
}

static void
bevy_run_context_layer_clear (BevyRunContextLayer *layer)
{
  g_assert (layer != NULL);
  g_assert (layer->qlink.data == layer);
  g_assert (layer->qlink.prev == NULL);
  g_assert (layer->qlink.next == NULL);

  if (layer->handler_data_destroy)
    g_clear_pointer (&layer->handler_data, layer->handler_data_destroy);

  g_clear_pointer (&layer->cwd, g_free);
  g_clear_pointer (&layer->argv, g_array_unref);
  g_clear_pointer (&layer->env, g_array_unref);
  g_clear_object (&layer->unix_fd_map);
}

static void
bevy_run_context_layer_free (BevyRunContextLayer *layer)
{
  bevy_run_context_layer_clear (layer);

  g_slice_free (BevyRunContextLayer, layer);
}

static void
strptr_free (gpointer data)
{
  char **strptr = data;
  g_clear_pointer (strptr, g_free);
}

static void
bevy_run_context_layer_init (BevyRunContextLayer *layer)
{
  g_assert (layer != NULL);

  layer->qlink.data = layer;
  layer->argv = g_array_new (TRUE, TRUE, sizeof (char *));
  layer->env = g_array_new (TRUE, TRUE, sizeof (char *));
  layer->unix_fd_map = bevy_unix_fd_map_new ();

  g_array_set_clear_func (layer->argv, strptr_free);
  g_array_set_clear_func (layer->env, strptr_free);
}

static BevyRunContextLayer *
bevy_run_context_current_layer (BevyRunContext *self)
{
  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (self->layers.length > 0);

  return self->layers.head->data;
}

static void
bevy_run_context_dispose (GObject *object)
{
  BevyRunContext *self = (BevyRunContext *)object;
  BevyRunContextLayer *layer;

  while ((layer = g_queue_peek_head (&self->layers)))
    {
      g_queue_unlink (&self->layers, &layer->qlink);
      if (layer != &self->root)
        bevy_run_context_layer_free (layer);
    }

  bevy_run_context_layer_clear (&self->root);

  G_OBJECT_CLASS (bevy_run_context_parent_class)->dispose (object);
}

static void
bevy_run_context_class_init (BevyRunContextClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->dispose = bevy_run_context_dispose;
}

static void
bevy_run_context_init (BevyRunContext *self)
{
  g_auto(GStrv) environ = g_get_environ ();

  self->setup_tty = TRUE;

  bevy_run_context_layer_init (&self->root);

  g_queue_push_head_link (&self->layers, &self->root.qlink);

  /* Always start with full environment at the base layer, but allow
   * removal by _set_environ(NULL) to clear it.
   */
  bevy_run_context_set_environ (self, (const char * const *)environ);
}

void
bevy_run_context_push (BevyRunContext        *self,
                         BevyRunContextHandler  handler,
                         gpointer                 handler_data,
                         GDestroyNotify           handler_data_destroy)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  layer = g_slice_new0 (BevyRunContextLayer);

  bevy_run_context_layer_init (layer);

  layer->handler = handler;
  layer->handler_data = handler_data;
  layer->handler_data_destroy = handler_data_destroy;

  g_queue_push_head_link (&self->layers, &layer->qlink);
}

void
bevy_run_context_push_at_base (BevyRunContext        *self,
                                 BevyRunContextHandler  handler,
                                 gpointer                 handler_data,
                                 GDestroyNotify           handler_data_destroy)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  layer = g_slice_new0 (BevyRunContextLayer);

  bevy_run_context_layer_init (layer);

  layer->handler = handler;
  layer->handler_data = handler_data;
  layer->handler_data_destroy = handler_data_destroy;

  _g_queue_insert_before_link (&self->layers, &self->root.qlink, &layer->qlink);
}

typedef struct
{
  char *shell;
  BevyRunContextShell kind : 2;
} Shell;

static void
shell_free (gpointer data)
{
  Shell *shell = data;
  g_clear_pointer (&shell->shell, g_free);
  g_slice_free (Shell, shell);
}

static gboolean
bevy_run_context_shell_handler (BevyRunContext    *self,
                                  const char * const  *argv,
                                  const char * const  *env,
                                  const char          *cwd,
                                  BevyUnixFDMap     *unix_fd_map,
                                  gpointer             user_data,
                                  GError             **error)
{
  Shell *shell = user_data;
  g_autoptr(GString) str = NULL;

  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (argv != NULL);
  g_assert (env != NULL);
  g_assert (BEVY_IS_UNIX_FD_MAP (unix_fd_map));
  g_assert (shell != NULL);
  g_assert (shell->shell != NULL);

  if (!bevy_run_context_merge_unix_fd_map (self, unix_fd_map, error))
    return FALSE;

  if (cwd != NULL)
    bevy_run_context_set_cwd (self, cwd);

  bevy_run_context_append_argv (self, shell->shell);
  if (shell->kind == BEVY_RUN_CONTEXT_SHELL_LOGIN)
    bevy_run_context_append_argv (self, "-l");
  else if (shell->kind == BEVY_RUN_CONTEXT_SHELL_INTERACTIVE)
    bevy_run_context_append_argv (self, "-i");
  bevy_run_context_append_argv (self, "-c");

  str = g_string_new (NULL);

  if (env[0] != NULL)
    {
      g_string_append (str, "env");

      for (guint i = 0; env[i]; i++)
        {
          g_autofree char *quoted = g_shell_quote (env[i]);

          g_string_append_c (str, ' ');
          g_string_append (str, quoted);
        }

      g_string_append_c (str, ' ');
    }

  for (guint i = 0; argv[i]; i++)
    {
      g_autofree char *quoted = g_shell_quote (argv[i]);

      if (i > 0)
        g_string_append_c (str, ' ');
      g_string_append (str, quoted);
    }

  bevy_run_context_append_argv (self, str->str);

  return TRUE;
}

/**
 * bevy_run_context_push_shell:
 * @self: a #BevyRunContext
 * @shell: the kind of shell to be used
 *
 * Pushes a shell which can run the upper layer command with -c.
 */
void
bevy_run_context_push_shell (BevyRunContext      *self,
                               BevyRunContextShell  shell)
{
  Shell *state;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  state = g_slice_new0 (Shell);
  state->shell = g_strdup ("/bin/sh");
  state->kind = shell;

  bevy_run_context_push (self, bevy_run_context_shell_handler, state, shell_free);
}

static gboolean
bevy_run_context_error_handler (BevyRunContext    *self,
                                  const char * const  *argv,
                                  const char * const  *env,
                                  const char          *cwd,
                                  BevyUnixFDMap     *unix_fd_map,
                                  gpointer             user_data,
                                  GError             **error)
{
  const GError *local_error = user_data;

  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (BEVY_IS_UNIX_FD_MAP (unix_fd_map));
  g_assert (local_error != NULL);

  if (error != NULL)
    *error = g_error_copy (local_error);

  return FALSE;
}

/**
 * bevy_run_context_push_error:
 * @self: a #BevyRunContext
 * @error: (transfer full) (in): a #GError
 *
 * Pushes a new layer that will always fail with @error.
 *
 * This is useful if you have an error when attempting to build
 * a run command, but need it to deliver the error when attempting
 * to create a subprocess launcher.
 */
void
bevy_run_context_push_error (BevyRunContext *self,
                               GError           *error)
{
  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (error != NULL);

  bevy_run_context_push (self,
                           bevy_run_context_error_handler,
                           error,
                           (GDestroyNotify)g_error_free);
}

const char * const *
bevy_run_context_get_argv (BevyRunContext *self)
{
  BevyRunContextLayer *layer;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);

  layer = bevy_run_context_current_layer (self);

  return (const char * const *)(gpointer)layer->argv->data;
}

void
bevy_run_context_set_argv (BevyRunContext   *self,
                             const char * const *argv)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  layer = bevy_run_context_current_layer (self);

  g_array_set_size (layer->argv, 0);

  if (argv != NULL)
    {
      char **copy = g_strdupv ((char **)argv);
      g_array_append_vals (layer->argv, copy, g_strv_length (copy));
      g_free (copy);
    }
}

const char * const *
bevy_run_context_get_environ (BevyRunContext *self)
{
  BevyRunContextLayer *layer;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);

  layer = bevy_run_context_current_layer (self);

  return (const char * const *)(gpointer)layer->env->data;
}

void
bevy_run_context_set_environ (BevyRunContext   *self,
                                const char * const *environ)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  layer = bevy_run_context_current_layer (self);

  g_array_set_size (layer->env, 0);

  if (environ != NULL && environ[0] != NULL)
    {
      char **copy = g_strdupv ((char **)environ);
      g_array_append_vals (layer->env, copy, g_strv_length (copy));
      g_free (copy);
    }
}

void
bevy_run_context_add_environ (BevyRunContext   *self,
                                const char * const *environ)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  if (environ == NULL || environ[0] == NULL)
    return;

  layer = bevy_run_context_current_layer (self);

  for (guint i = 0; environ[i]; i++)
    {
      const char *pair = environ[i];
      const char *eq = strchr (pair, '=');
      char **dest = NULL;
      gsize keylen;

      if (eq == NULL)
        continue;

      keylen = eq - pair;

      for (guint j = 0; j < layer->env->len; j++)
        {
          const char *ele = g_array_index (layer->env, const char *, j);

          if (strncmp (pair, ele, keylen) == 0 && ele[keylen] == '=')
            {
              dest = &g_array_index (layer->env, char *, j);
              break;
            }
        }

      if (dest == NULL)
        {
          g_array_set_size (layer->env, layer->env->len + 1);
          dest = &g_array_index (layer->env, char *, layer->env->len - 1);
        }

      g_clear_pointer (dest, g_free);
      *dest = g_strdup (pair);
    }
}

const char *
bevy_run_context_get_cwd (BevyRunContext *self)
{
  BevyRunContextLayer *layer;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);

  layer = bevy_run_context_current_layer (self);

  return layer->cwd;
}

void
bevy_run_context_set_cwd (BevyRunContext *self,
                            const char       *cwd)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  layer = bevy_run_context_current_layer (self);

  _g_set_str (&layer->cwd, cwd);
}

void
bevy_run_context_prepend_argv (BevyRunContext *self,
                                 const char       *arg)
{
  BevyRunContextLayer *layer;
  char *copy;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (arg != NULL);

  layer = bevy_run_context_current_layer (self);

  copy = g_strdup (arg);
  g_array_insert_val (layer->argv, 0, copy);
}

void
bevy_run_context_prepend_args (BevyRunContext   *self,
                                 const char * const *args)
{
  BevyRunContextLayer *layer;
  char **copy;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  if (args == NULL || args[0] == NULL)
    return;

  layer = bevy_run_context_current_layer (self);

  copy = g_strdupv ((char **)args);
  g_array_insert_vals (layer->argv, 0, copy, g_strv_length (copy));
  g_free (copy);
}

void
bevy_run_context_append_argv (BevyRunContext *self,
                                const char       *arg)
{
  BevyRunContextLayer *layer;
  char *copy;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (arg != NULL);

  layer = bevy_run_context_current_layer (self);

  copy = g_strdup (arg);
  g_array_append_val (layer->argv, copy);
}

void
bevy_run_context_append_formatted (BevyRunContext *self,
                                     const char       *format,
                                     ...)
{
  g_autofree char *arg = NULL;
  va_list args;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (format != NULL);

  va_start (args, format);
  arg = g_strdup_vprintf (format, args);
  va_end (args);

  bevy_run_context_append_argv (self, arg);
}

void
bevy_run_context_append_args (BevyRunContext   *self,
                                const char * const *args)
{
  BevyRunContextLayer *layer;
  char **copy;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  if (args == NULL || args[0] == NULL)
    return;

  layer = bevy_run_context_current_layer (self);

  copy = g_strdupv ((char **)args);
  g_array_append_vals (layer->argv, copy, g_strv_length (copy));
  g_free (copy);
}

gboolean
bevy_run_context_append_args_parsed (BevyRunContext  *self,
                                       const char        *args,
                                       GError           **error)
{
  BevyRunContextLayer *layer;
  char **argv = NULL;
  int argc;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), FALSE);
  g_return_val_if_fail (args != NULL, FALSE);

  layer = bevy_run_context_current_layer (self);

  if (!g_shell_parse_argv (args, &argc, &argv, error))
    return FALSE;

  g_array_append_vals (layer->argv, argv, argc);
  g_free (argv);

  return TRUE;
}

void
bevy_run_context_take_fd (BevyRunContext *self,
                            int               source_fd,
                            int               dest_fd)
{
  BevyRunContextLayer *layer;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (source_fd >= -1);
  g_return_if_fail (dest_fd > -1);

  layer = bevy_run_context_current_layer (self);

  bevy_unix_fd_map_take (layer->unix_fd_map, source_fd, dest_fd);
}

const char *
bevy_run_context_getenv (BevyRunContext *self,
                           const char       *key)
{
  BevyRunContextLayer *layer;
  gsize keylen;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);
  g_return_val_if_fail (key != NULL, NULL);

  layer = bevy_run_context_current_layer (self);

  keylen = strlen (key);

  for (guint i = 0; i < layer->env->len; i++)
    {
      const char *envvar = g_array_index (layer->env, const char *, i);

      if (strncmp (key, envvar, keylen) == 0 && envvar[keylen] == '=')
        return &envvar[keylen+1];
    }

  return NULL;
}

void
bevy_run_context_setenv (BevyRunContext *self,
                           const char       *key,
                           const char       *value)
{
  BevyRunContextLayer *layer;
  char *element;
  gsize keylen;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (key != NULL);

  if (value == NULL)
    {
      bevy_run_context_unsetenv (self, key);
      return;
    }

  layer = bevy_run_context_current_layer (self);

  keylen = strlen (key);
  element = g_strconcat (key, "=", value, NULL);

  g_array_append_val (layer->env, element);

  for (guint i = 0; i < layer->env->len-1; i++)
    {
      const char *envvar = g_array_index (layer->env, const char *, i);

      if (strncmp (key, envvar, keylen) == 0 && envvar[keylen] == '=')
        {
          g_array_remove_index_fast (layer->env, i);
          break;
        }
    }
}

void
bevy_run_context_unsetenv (BevyRunContext *self,
                             const char       *key)
{
  BevyRunContextLayer *layer;
  gsize len;

  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));
  g_return_if_fail (key != NULL);

  layer = bevy_run_context_current_layer (self);

  len = strlen (key);

  for (guint i = 0; i < layer->env->len; i++)
    {
      const char *envvar = g_array_index (layer->env, const char *, i);

      if (strncmp (key, envvar, len) == 0 && envvar[len] == '=')
        {
          g_array_remove_index_fast (layer->env, i);
          return;
        }
    }
}

void
bevy_run_context_environ_to_argv (BevyRunContext *self)
{
  BevyRunContextLayer *layer;
  const char **copy;

  g_assert (BEVY_IS_RUN_CONTEXT (self));

  layer = bevy_run_context_current_layer (self);

  if (layer->env->len == 0)
    return;

  copy = (const char **)g_new0 (char *, layer->env->len + 2);
  copy[0] = "env";
  for (guint i = 0; i < layer->env->len; i++)
    copy[1+i] = g_array_index (layer->env, const char *, i);
  bevy_run_context_prepend_args (self, (const char * const *)copy);
  g_free (copy);

  g_array_set_size (layer->env, 0);
}

static gboolean
bevy_run_context_default_handler (BevyRunContext    *self,
                                    const char * const  *argv,
                                    const char * const  *env,
                                    const char          *cwd,
                                    BevyUnixFDMap     *unix_fd_map,
                                    gpointer             user_data,
                                    GError             **error)
{
  BevyRunContextLayer *layer;

  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (argv != NULL);
  g_assert (env != NULL);
  g_assert (BEVY_IS_UNIX_FD_MAP (unix_fd_map));

  layer = bevy_run_context_current_layer (self);

  if (cwd != NULL)
    {
      /* If the working directories do not match, we can't satisfy this and
       * need to error out.
       */
      if (layer->cwd != NULL && !g_str_equal (cwd, layer->cwd))
        {
          g_set_error (error,
                       G_IO_ERROR,
                       G_IO_ERROR_INVALID_ARGUMENT,
                       "Cannot resolve differently requested cwd: %s and %s",
                       cwd, layer->cwd);
          return FALSE;
        }

      bevy_run_context_set_cwd (self, cwd);
    }

  /* Merge all the FDs unless there are collisions */
  if (!bevy_unix_fd_map_steal_from (layer->unix_fd_map, unix_fd_map, error))
    return FALSE;

  if (env[0] != NULL)
    {
      if (argv[0] == NULL)
        {
          bevy_run_context_add_environ (self, env);
        }
      else
        {
          bevy_run_context_append_argv (self, "env");
          bevy_run_context_append_args (self, env);
        }
    }

  if (argv[0] != NULL)
    bevy_run_context_append_args (self, argv);

  return TRUE;
}

static int
sort_strptr (gconstpointer a,
             gconstpointer b)
{
  const char * const *astr = a;
  const char * const *bstr = b;

  return g_strcmp0 (*astr, *bstr);
}

static gboolean
bevy_run_context_callback_layer (BevyRunContext       *self,
                                   BevyRunContextLayer  *layer,
                                   GError                **error)
{
  BevyRunContextHandler handler;
  gpointer handler_data;
  gboolean ret;

  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (layer != NULL);
  g_assert (layer != &self->root);

  handler = layer->handler ? layer->handler : bevy_run_context_default_handler;
  handler_data = layer->handler ? layer->handler_data : NULL;

  /* Sort environment variables first so that we have an easier time
   * finding them by eye in tooling which translates them.
   */
  g_array_sort (layer->env, sort_strptr);

  ret = handler (self,
                 (const char * const *)(gpointer)layer->argv->data,
                 (const char * const *)(gpointer)layer->env->data,
                 layer->cwd,
                 layer->unix_fd_map,
                 handler_data,
                 error);

  bevy_run_context_layer_free (layer);

  return ret;
}

static void
set_max_fd_limit (gint64 value)
{
  struct rlimit limit;

  if (value <= 0)
    return;

  if (getrlimit (RLIMIT_NOFILE, &limit) == 0)
    {
      limit.rlim_cur = value;

      if (setrlimit (RLIMIT_NOFILE, &limit) != 0)
        g_warning ("Failed to set FD limit to %"G_GSSIZE_FORMAT"",
                   (gssize)value);
    }
}

static void
bevy_run_context_child_setup_cb (gpointer data)
{
  gboolean setup_tty = GPOINTER_TO_INT (data);

  set_max_fd_limit (bevy_agent_get_default_rlimit_nofile ());

  setsid ();
  setpgid (0, 0);

#ifdef __linux__
  prctl (PR_SET_PDEATHSIG, SIGHUP);
#endif

  if (setup_tty)
    {
      if (isatty (STDIN_FILENO))
        ioctl (STDIN_FILENO, TIOCSCTTY, 0);
    }
}

/**
 * bevy_run_context_spawn:
 * @self: a #BevyRunContext
 *
 * Returns: (transfer full): an #GSubprocessLauncher if successful; otherwise
 *   %NULL and @error is set.
 */
GSubprocess *
bevy_run_context_spawn (BevyRunContext  *self,
                          GError           **error)
{
  return bevy_run_context_spawn_with_flags (self, 0, error);
}

GSubprocess *
bevy_run_context_spawn_with_flags (BevyRunContext  *self,
                                     GSubprocessFlags   flags,
                                     GError           **error)
{
  g_autoptr(GSubprocessLauncher) launcher = NULL;
  const char * const *argv;
  guint length;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);
  g_return_val_if_fail (self->ended == FALSE, NULL);

  self->ended = TRUE;

  while (self->layers.length > 1)
    {
      BevyRunContextLayer *layer = bevy_run_context_current_layer (self);

      g_queue_unlink (&self->layers, &layer->qlink);

      if (!bevy_run_context_callback_layer (self, layer, error))
        return FALSE;
    }

  argv = bevy_run_context_get_argv (self);

  launcher = g_subprocess_launcher_new (0);

  g_subprocess_launcher_set_environ (launcher, (char **)bevy_run_context_get_environ (self));
  g_subprocess_launcher_set_cwd (launcher, bevy_run_context_get_cwd (self));

  length = bevy_unix_fd_map_get_length (self->root.unix_fd_map);

  for (guint i = 0; i < length; i++)
    {
      int source_fd;
      int dest_fd;

      source_fd = bevy_unix_fd_map_steal (self->root.unix_fd_map, i, &dest_fd);

      if (dest_fd == STDOUT_FILENO && source_fd == -1 && (flags & G_SUBPROCESS_FLAGS_STDOUT_PIPE) == 0)
        flags |= G_SUBPROCESS_FLAGS_STDOUT_SILENCE;

      if (dest_fd == STDERR_FILENO && source_fd == -1 && (flags & G_SUBPROCESS_FLAGS_STDERR_PIPE) == 0)
        flags |= G_SUBPROCESS_FLAGS_STDERR_SILENCE;

      if (source_fd != -1 && dest_fd != -1)
        {
          if (dest_fd == STDIN_FILENO)
            g_subprocess_launcher_take_stdin_fd (launcher, source_fd);
          else if (dest_fd == STDOUT_FILENO)
            g_subprocess_launcher_take_stdout_fd (launcher, source_fd);
          else if (dest_fd == STDERR_FILENO)
            g_subprocess_launcher_take_stderr_fd (launcher, source_fd);
          else
            g_subprocess_launcher_take_fd (launcher, source_fd, dest_fd);
        }
    }

  g_subprocess_launcher_set_flags (launcher, flags);
  g_subprocess_launcher_set_child_setup (launcher,
                                         bevy_run_context_child_setup_cb,
                                         GINT_TO_POINTER (self->setup_tty),
                                         NULL);

  return g_subprocess_launcher_spawnv (launcher, argv, error);
}

/**
 * bevy_run_context_merge_unix_fd_map:
 * @self: a #BevyRunContext
 * @unix_fd_map: a #BevyUnixFDMap
 * @error: a #GError, or %NULL
 *
 * Merges the #BevyUnixFDMap into the current layer.
 *
 * If there are collisions in destination FDs, then that may cause an
 * error and %FALSE is returned.
 *
 * @unix_fd_map will have the FDs stolen using bevy_unix_fd_map_steal_from()
 * which means that if successful, @unix_fd_map will not have any open
 * file-descriptors after calling this function.
 *
 * Returns: %TRUE if successful; otherwise %FALSE and @error is set.
 */
gboolean
bevy_run_context_merge_unix_fd_map (BevyRunContext  *self,
                                      BevyUnixFDMap   *unix_fd_map,
                                      GError           **error)
{
  BevyRunContextLayer *layer;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), FALSE);
  g_return_val_if_fail (BEVY_IS_UNIX_FD_MAP (unix_fd_map), FALSE);

  layer = bevy_run_context_current_layer (self);

  return bevy_unix_fd_map_steal_from (layer->unix_fd_map, unix_fd_map, error);
}

/**
 * bevy_run_context_create_stdio_stream:
 * @self: a #BevyRunContext
 * @error: a location for a #GError
 *
 * Creates a stream to communicate with the subprocess using stdin/stdout.
 *
 * The stream is created using UNIX pipes which are attached to the
 * stdin/stdout of the child process.
 *
 * Returns: (transfer full): a #GIOStream if successful; otherwise
 *   %NULL and @error is set.
 */
GIOStream *
bevy_run_context_create_stdio_stream (BevyRunContext  *self,
                                        GError           **error)
{
  BevyRunContextLayer *layer;

  g_return_val_if_fail (BEVY_IS_RUN_CONTEXT (self), NULL);

  layer = bevy_run_context_current_layer (self);

  return bevy_unix_fd_map_create_stream (layer->unix_fd_map,
                                           STDIN_FILENO,
                                           STDOUT_FILENO,
                                           error);
}

static gboolean
has_systemd (void)
{
  static gboolean initialized;
  static gboolean has_systemd;

  if (!initialized)
    {
      g_autofree char *path = g_find_program_in_path ("systemd-run");
      g_autofree char *stdout_str = NULL;
      int wait_status = -1;

      if (path != NULL &&
          g_spawn_sync (NULL,
                        (char **)(const char * const []) {path, "--version", NULL},
                        NULL,
                        G_SPAWN_STDERR_TO_DEV_NULL,
                        NULL,
                        NULL,
                        &stdout_str,
                        NULL,
                        &wait_status,
                        NULL) &&
          stdout_str != NULL &&
          g_str_has_prefix (stdout_str, "systemd "))
        {
          const char *str = stdout_str + strlen ("systemd ");
          int systemd_version = atoi (str);

          /* We require systemd-run 240 for --same-dir/--working-directory but also
           * because it doesn't seem to work at all on CentOS 7 which is 219.
           */
          has_systemd = systemd_version >= 240;
        }

      initialized = TRUE;
    }

  return has_systemd;
}

static gboolean
bevy_run_context_push_scope_cb (BevyRunContext    *self,
                                  const char * const  *argv,
                                  const char * const  *env,
                                  const char          *cwd,
                                  BevyUnixFDMap     *unix_fd_map,
                                  gpointer             user_data,
                                  GError             **error)
{
  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (BEVY_IS_UNIX_FD_MAP (unix_fd_map));

  if (!bevy_run_context_merge_unix_fd_map (self, unix_fd_map, error))
    return FALSE;

  bevy_run_context_set_cwd (self, cwd);
  bevy_run_context_set_environ (self, env);

  if (has_systemd ())
    {
      g_autofree char *uuid = g_uuid_string_random ();

      bevy_run_context_append_argv (self, "systemd-run");
      bevy_run_context_append_argv (self, "--user");
      bevy_run_context_append_argv (self, "--scope");
      bevy_run_context_append_argv (self, "--collect");
      bevy_run_context_append_argv (self, "--quiet");
      bevy_run_context_append_argv (self, "--same-dir");
      bevy_run_context_append_formatted (self, "--unit=bevy-spawn-%s.scope", uuid);
    }

  bevy_run_context_append_args (self, argv);

  return TRUE;
}

void
bevy_run_context_push_scope (BevyRunContext *self)
{
  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  bevy_run_context_push (self,
                           bevy_run_context_push_scope_cb,
                           NULL, NULL);
}

static gboolean
bevy_run_context_host_handler (BevyRunContext    *self,
                                 const char * const  *argv,
                                 const char * const  *env,
                                 const char          *cwd,
                                 BevyUnixFDMap     *unix_fd_map,
                                 gpointer             user_data,
                                 GError             **error)
{
  guint length;

  g_assert (BEVY_IS_RUN_CONTEXT (self));
  g_assert (argv != NULL);
  g_assert (env != NULL);
  g_assert (BEVY_IS_UNIX_FD_MAP (unix_fd_map));
  g_assert (bevy_agent_is_sandboxed ());

  bevy_run_context_append_argv (self, "flatpak-spawn");
  bevy_run_context_append_argv (self, "--host");
  bevy_run_context_append_argv (self, "--watch-bus");

  if (env != NULL)
    {
      for (guint i = 0; env[i]; i++)
        bevy_run_context_append_formatted (self, "--env=%s", env[i]);
    }

  if (cwd != NULL)
    bevy_run_context_append_formatted (self, "--directory=%s", cwd);

  if ((length = bevy_unix_fd_map_get_length (unix_fd_map)))
    {
      for (guint i = 0; i < length; i++)
        {
          int source_fd;
          int dest_fd;

          source_fd = bevy_unix_fd_map_peek (unix_fd_map, i, &dest_fd);

          if (dest_fd < STDERR_FILENO)
            continue;

          g_debug ("Mapping Builder FD %d to target FD %d via flatpak-spawn",
                   source_fd, dest_fd);

          if (source_fd != -1 && dest_fd != -1)
            bevy_run_context_append_formatted (self, "--forward-fd=%d", dest_fd);
        }

      if (!bevy_run_context_merge_unix_fd_map (self, unix_fd_map, error))
        return FALSE;
    }

  /* Now append the arguments */
  bevy_run_context_append_args (self, argv);

  return TRUE;
}

void
bevy_run_context_push_host (BevyRunContext *self)
{
  g_return_if_fail (BEVY_IS_RUN_CONTEXT (self));

  if (!bevy_agent_is_sandboxed ())
    return;

  self->setup_tty = FALSE;

  bevy_run_context_push (self,
                           bevy_run_context_host_handler,
                           NULL, NULL);
}
