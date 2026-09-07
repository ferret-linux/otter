/*
 * bevy-tab.c
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

#include "config.h"

#include <glib/gi18n.h>

#include <cairo.h>

#ifdef __linux__
# include <libportal/portal.h>
# include <libportal-gtk4/portal-gtk4.h>
#endif

#include "bevy-agent-ipc.h"
#include "bevy-application.h"
#include "bevy-enums.h"
#include "bevy-inspector.h"
#include "bevy-tab-monitor.h"
#include "bevy-tab-notify.h"
#include "bevy-tab-private.h"
#include "bevy-terminal.h"
#include "bevy-util.h"
#include "bevy-window.h"

typedef enum _BevyTabState
{
  BEVY_TAB_STATE_INITIAL,
  BEVY_TAB_STATE_SPAWNING,
  BEVY_TAB_STATE_RUNNING,
  BEVY_TAB_STATE_EXITED,
  BEVY_TAB_STATE_FAILED,
} BevyTabState;

struct _BevyTab
{
  GtkWidget                parent_instance;

  char                    *initial_working_directory_uri;
  char                    *previous_working_directory_uri;
  BevyProfile           *profile;
  BevyIpcProcess        *process;
  char                    *title_prefix;
  BevyTabMonitor        *monitor;
  char                    *uuid;
  BevyIpcContainer      *container_at_creation;
  char                   **command;
  char                    *initial_title;
  GdkTexture              *cached_texture;
  GtkCssProvider          *css_provider;
  AdwBanner               *banner;
  GtkScrolledWindow       *scrolled_window;
  BevyTerminal          *terminal;
  char                    *command_line;
  char                    *program_name;
  BevyTabNotify          notify;
  GSignalGroup            *profile_signals;

  BevyTabState           state;
  GPid                     pid;

  gint64                   respawn_time;

  BevyZoomLevel          zoom : 5;
  BevyProcessLeaderKind  leader_kind : 3;
  guint                    has_foreground_process : 1;
  guint                    forced_exit : 1;
  guint                    ignore_osc_title : 1;
  guint                    ignore_snapshot : 1;

  guint                    inhibit_cookie;
};

enum {
  PROP_0,
  PROP_COMMAND_LINE,
  PROP_ICON,
  PROP_IGNORE_OSC_TITLE,
  PROP_INDICATOR_ICON,
  PROP_PROCESS_LEADER_KIND,
  PROP_PROFILE,
  PROP_PROGRESS,
  PROP_PROGRESS_FRACTION,
  PROP_READ_ONLY,
  PROP_SUBTITLE,
  PROP_TITLE,
  PROP_TITLE_PREFIX,
  PROP_UUID,
  PROP_ZOOM,
  PROP_ZOOM_LABEL,
  N_PROPS
};

enum {
  BELL,
  COMMIT,
  N_SIGNALS
};

static void bevy_tab_respawn (BevyTab *self);
static void bevy_tab_profile_signals_bind_cb (BevyTab     *self,
                                                BevyProfile *profile,
                                                GSignalGroup  *group);

G_DEFINE_FINAL_TYPE (BevyTab, bevy_tab, GTK_TYPE_WIDGET)

#ifdef __linux__
static XdpPortal *portal;
#endif

static GParamSpec *properties[N_PROPS];
static guint signals[N_SIGNALS];
static double zoom_font_scales[] = {
  0,

  /* MINUS_14 through MINUS_1: each step is 1.2^(1/2) ≈ 1.095445 */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2),                     /* MINUS_14: 1.2^(-7) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2) * 1.095445115010332, /* MINUS_13: 1.2^(-6.5) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2),                           /* MINUS_12: 1.2^(-6) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2) * 1.095445115010332,       /* MINUS_11: 1.2^(-5.5) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2),                                 /* MINUS_10: 1.2^(-5) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2 * 1.2) * 1.095445115010332,             /* MINUS_9: 1.2^(-4.5) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2),                                       /* MINUS_8: 1.2^(-4) */
  1.0 / (1.2 * 1.2 * 1.2 * 1.2) * 1.095445115010332,                   /* MINUS_7: 1.2^(-3.5) */
  1.0 / (1.2 * 1.2 * 1.2),                                             /* MINUS_6: 1.2^(-3) */
  1.0 / (1.2 * 1.2 * 1.2) * 1.095445115010332,                         /* MINUS_5: 1.2^(-2.5) */
  1.0 / (1.2 * 1.2),                                                   /* MINUS_4: 1.2^(-2) */
  1.0 / (1.2 * 1.2) * 1.095445115010332,                               /* MINUS_3: 1.2^(-1.5) */
  1.0 / (1.2),                                                         /* MINUS_2: 1.2^(-1) */
  1.0 / (1.2) * 1.095445115010332,                                     /* MINUS_1: 1.2^(-0.5) */
  1.0,                                                                 /* DEFAULT: 1.2^0 */

  /* PLUS_1 through PLUS_14: each step is 1.2^(1/2) ≈ 1.095445 */
  1.0 * 1.095445115010332,                                             /* PLUS_1: 1.2^0.5 */
  1.0 * 1.2,                                                           /* PLUS_2: 1.2^1 */
  1.0 * 1.2 * 1.095445115010332,                                       /* PLUS_3: 1.2^1.5 */
  1.0 * 1.2 * 1.2,                                                     /* PLUS_4: 1.2^2 */
  1.0 * 1.2 * 1.2 * 1.095445115010332,                                 /* PLUS_5: 1.2^2.5 */
  1.0 * 1.2 * 1.2 * 1.2,                                               /* PLUS_6: 1.2^3 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.095445115010332,                           /* PLUS_7: 1.2^3.5 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2,                                         /* PLUS_8: 1.2^4 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.095445115010332,                     /* PLUS_9: 1.2^4.5 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2,                                   /* PLUS_10: 1.2^5 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.095445115010332,               /* PLUS_11: 1.2^5.5 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2,                             /* PLUS_12: 1.2^6 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.095445115010332,         /* PLUS_13: 1.2^6.5 */
  1.0 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2 * 1.2,                       /* PLUS_14: 1.2^7 */
};

static gboolean
on_scroll_scrolled_cb (GtkEventControllerScroll *scroll,
                       double                    dx,
                       double                    dy,
                       BevyTab                *self)
{
  GdkModifierType mods;

  g_assert (GTK_IS_EVENT_CONTROLLER_SCROLL (scroll));
  g_assert (BEVY_IS_TAB (self));

  mods = gtk_event_controller_get_current_event_state (GTK_EVENT_CONTROLLER (scroll));

  if ((mods & GDK_CONTROL_MASK) != 0)
    {
      BevySettings *settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);

      if (bevy_settings_get_enable_zoom_scroll_ctrl(settings))
        {
          if (dy < 0)
            bevy_tab_zoom_in (self);
          else if (dy > 0)
            bevy_tab_zoom_out (self);
	}

      return TRUE;
    }

  return FALSE;
}

static void
on_scroll_begin_cb (GtkEventControllerScroll *scroll,
                    BevyTab                *self)
{
  GdkModifierType state;

  g_assert (GTK_IS_EVENT_CONTROLLER_SCROLL (scroll));
  g_assert (BEVY_IS_TAB (self));

  state = gtk_event_controller_get_current_event_state (GTK_EVENT_CONTROLLER (scroll));

  if ((state & GDK_CONTROL_MASK) != 0)
    gtk_event_controller_scroll_set_flags (scroll,
                                           GTK_EVENT_CONTROLLER_SCROLL_VERTICAL |
                                           GTK_EVENT_CONTROLLER_SCROLL_DISCRETE);
}

static void
on_scroll_end_cb (GtkEventControllerScroll *scroll,
                  BevyTab                *self)
{
  g_assert (GTK_IS_EVENT_CONTROLLER_SCROLL (scroll));
  g_assert (BEVY_IS_TAB (self));

  gtk_event_controller_scroll_set_flags (scroll, GTK_EVENT_CONTROLLER_SCROLL_VERTICAL);
}

static void
bevy_tab_send_signal (BevyTab *self,
                        int        signum)
{
  g_autofree char *title = NULL;

  g_assert (BEVY_IS_TAB (self));

  if (self->process == NULL)
    {
      g_debug ("Cannot send signal %d to tab, process is gone.", signum);
      return;
    }

  title = bevy_tab_dup_title (self);
  g_debug ("Sending signal %d to tab \"%s\"", signum, title);

  bevy_ipc_process_call_send_signal (self->process, signum, NULL, NULL, NULL);
}

static gboolean
bevy_tab_is_active (BevyTab *self)
{
  GtkWidget *window;

  g_assert (BEVY_IS_TAB (self));

  if ((window = gtk_widget_get_ancestor (GTK_WIDGET (self), BEVY_TYPE_WINDOW)))
    return bevy_window_get_active_tab (BEVY_WINDOW (window)) == self;

  return FALSE;
}

static void
bevy_tab_update_scrollback_lines (BevyTab *self)
{
  long scrollback_lines = -1;

  g_assert (BEVY_IS_TAB (self));

  if (bevy_profile_get_limit_scrollback (self->profile))
    scrollback_lines = bevy_profile_get_scrollback_lines (self->profile);

  vte_terminal_set_scrollback_lines (VTE_TERMINAL (self->terminal), scrollback_lines);
}

static void
bevy_tab_update_cell_height_scale (BevyTab *self)
{
  double cell_height_scale = 1.0;

  g_assert (BEVY_IS_TAB (self));

  if (bevy_profile_get_cell_height_scale (self->profile))
    cell_height_scale = bevy_profile_get_cell_height_scale (self->profile);

  vte_terminal_set_cell_height_scale (VTE_TERMINAL (self->terminal), cell_height_scale);
}

static void
bevy_tab_update_cell_width_scale (BevyTab *self)
{
  double cell_width_scale = 1.0;

  g_assert (BEVY_IS_TAB (self));

  if (bevy_profile_get_cell_width_scale (self->profile))
    cell_width_scale = bevy_profile_get_cell_width_scale (self->profile);

  vte_terminal_set_cell_width_scale (VTE_TERMINAL (self->terminal), cell_width_scale);
}

static void
bevy_tab_update_margins (BevyTab *self)
{
  g_autofree char *css = NULL;
  int margin_x, margin_y;

  g_assert (BEVY_IS_TAB (self));

  margin_x = bevy_profile_get_margin_x (self->profile);
  margin_y = bevy_profile_get_margin_y (self->profile);

  css = g_strdup_printf (".bevy-tab-%s { padding: %dpx %dpx; }",
                         self->uuid, margin_y, margin_x);
  gtk_css_provider_load_from_string (self->css_provider, css);
}

static void
bevy_tab_update_custom_links (BevyTab *self)
{
  g_autoptr(GListModel) custom_links_list = NULL;

  g_assert (BEVY_IS_TAB (self));

  custom_links_list = bevy_profile_list_custom_links(self->profile);
  bevy_terminal_update_custom_links_list(self->terminal, custom_links_list);
}

static void
bevy_tab_update_inhibit (BevyTab *self)
{
  BevySettings *settings;
  gboolean inhibit = FALSE;
  GtkWidget *window;

  g_assert (BEVY_IS_TAB (self));

  settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);

  /* Clear if the user has disabled logout inhibition */
  if (!bevy_settings_get_inhibit_logout (settings))
    {
      if (self->inhibit_cookie)
        {
          gtk_application_uninhibit (GTK_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                     self->inhibit_cookie);
          self->inhibit_cookie = 0;
        }

      return;
    }

  /* Only inhibit if there's a foreground process running and it's not a shell */
  if (self->has_foreground_process &&
      self->program_name != NULL &&
      !bevy_is_shell (self->program_name))
    inhibit = TRUE;

  /* Check if we need to change the inhibit state */
  if ((inhibit && self->inhibit_cookie != 0) ||
      (!inhibit && self->inhibit_cookie == 0))
    return;

  /* Get the window to use for the inhibit call */
  window = gtk_widget_get_ancestor (GTK_WIDGET (self), GTK_TYPE_WINDOW);

  if (inhibit)
    {
      /* Only inhibit if we have a valid window reference */
      if (window != NULL)
        {
          self->inhibit_cookie =
            gtk_application_inhibit (GTK_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                     GTK_WINDOW (window),
                                     GTK_APPLICATION_INHIBIT_LOGOUT,
                                     _("A foreground process is running"));
        }
    }
  else
    {
      gtk_application_uninhibit (GTK_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                 self->inhibit_cookie);
      self->inhibit_cookie = 0;
    }
}

static void
bevy_tab_wait_cb (GObject      *object,
                    GAsyncResult *result,
                    gpointer      user_data)
{
  BevyApplication *app = (BevyApplication *)object;
  g_autoptr(BevyTab) self = user_data;
  g_autoptr(GError) error = NULL;
  BevyExitAction exit_action;
  BevyWindow *window;
  AdwTabPage *page = NULL;
  GtkWidget *tab_view;
  gboolean is_front = FALSE;
  int exit_code;

  g_assert (BEVY_IS_APPLICATION (app));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (BEVY_IS_TAB (self));
  g_assert (self->state == BEVY_TAB_STATE_RUNNING);

  g_clear_object (&self->process);

  /* Update inhibit state when process exits */
  bevy_tab_update_inhibit (self);

  exit_code = bevy_application_wait_finish (app, result, &error);

  g_debug ("Process completed with exit-code 0x%x %s",
           exit_code,
           error ? error->message : "");

  if (error == NULL && WIFEXITED (exit_code) && WEXITSTATUS (exit_code) == 0)
    self->state = BEVY_TAB_STATE_EXITED;
  else
    self->state = BEVY_TAB_STATE_FAILED;

  if (self->forced_exit)
    return;

  if ((window = BEVY_WINDOW (gtk_widget_get_ancestor (GTK_WIDGET (self), BEVY_TYPE_WINDOW))))
    is_front = self == bevy_window_get_active_tab (window);

  if (WIFSIGNALED (exit_code))
    {
      g_autofree char *title = NULL;

      title = g_strdup_printf (_("Process Exited from Signal %d"), WTERMSIG (exit_code));

      adw_banner_set_title (self->banner, title);
      adw_banner_set_button_label (self->banner, _("_Restart"));
      gtk_actionable_set_action_name (GTK_ACTIONABLE (self->banner), "tab.respawn");
      gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);
      return;
    }

  exit_action = bevy_profile_get_exit_action (self->profile);
  tab_view = gtk_widget_get_ancestor (GTK_WIDGET (self), ADW_TYPE_TAB_VIEW);

  /* If this was started with something like bevy_window_new_for_command()
   * then we just want to exit the application (so allow tab to close).
   */
  if (self->command != NULL)
    exit_action = BEVY_EXIT_ACTION_CLOSE;

  if (ADW_IS_TAB_VIEW (tab_view))
    page = adw_tab_view_get_page (ADW_TAB_VIEW (tab_view), GTK_WIDGET (self));

  /* Always prepare the banner even if we don't show it because we may
   * display it again if the tab is removed from the parking lot and
   * restored into the window.
   */
  adw_banner_set_title (self->banner, _("Process Exited"));
  adw_banner_set_button_label (self->banner, _("_Restart"));
  gtk_actionable_set_action_name (GTK_ACTIONABLE (self->banner), "tab.respawn");

  /* If we took less than .5 a second to spawn and no key has been
   * pressed in the terminal, then treat this as a failed spawn. Don't
   * allow ourselves to auto-close in that case as it's likely an error
   * the user would want to see.
   */
  if ((self->command == NULL || self->state == BEVY_TAB_STATE_FAILED) &&
      (g_get_monotonic_time () - self->respawn_time) < (G_USEC_PER_SEC/2) &&
      !bevy_tab_monitor_get_has_pressed_key (self->monitor))
    exit_action = BEVY_EXIT_ACTION_NONE;

  switch (exit_action)
    {
    case BEVY_EXIT_ACTION_RESTART:
      bevy_tab_respawn (self);
      break;

    case BEVY_EXIT_ACTION_CLOSE:
      if (ADW_IS_TAB_VIEW (tab_view) && ADW_IS_TAB_PAGE (page))
        {
          if (adw_tab_page_get_pinned (page))
            adw_tab_view_set_page_pinned (ADW_TAB_VIEW (tab_view), page, FALSE);
          adw_tab_view_close_page (ADW_TAB_VIEW (tab_view), page);
        }
      break;

    case BEVY_EXIT_ACTION_NONE:
      gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);
      if (is_front)
        gtk_widget_child_focus (GTK_WIDGET (self->banner), GTK_DIR_TAB_FORWARD);
      break;

    default:
      g_assert_not_reached ();
    }

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);
}

static void
bevy_tab_spawn_cb (GObject      *object,
                     GAsyncResult *result,
                     gpointer      user_data)
{
  BevyApplication *app = (BevyApplication *)object;
  g_autoptr(BevyIpcProcess) process = NULL;
  g_autoptr(BevyTab) self = user_data;
  g_autoptr(GError) error = NULL;

  g_assert (BEVY_IS_TAB (self));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (BEVY_IS_TAB (self));
  g_assert (self->state == BEVY_TAB_STATE_SPAWNING);

  if (!(process = bevy_application_spawn_finish (app, result, &error)))
    {
      const char *profile_uuid = bevy_profile_get_uuid (self->profile);

      self->state = BEVY_TAB_STATE_FAILED;

      vte_terminal_feed (VTE_TERMINAL (self->terminal), error->message, -1);
      vte_terminal_feed (VTE_TERMINAL (self->terminal), "\r\n", -1);

      adw_banner_set_title (self->banner, _("Failed to launch terminal"));
      adw_banner_set_button_label (self->banner, _("Edit Profile"));
      gtk_actionable_set_action_target (GTK_ACTIONABLE (self->banner), "s", profile_uuid);
      gtk_actionable_set_action_name (GTK_ACTIONABLE (self->banner), "app.edit-profile");
      gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);

      return;
    }

  self->state = BEVY_TAB_STATE_RUNNING;
  self->respawn_time = g_get_monotonic_time ();

  g_set_object (&self->process, process);

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ICON]);

  bevy_application_wait_async (app,
                                 process,
                                 NULL,
                                 bevy_tab_wait_cb,
                                 g_object_ref (self));
}

static void
bevy_tab_respawn (BevyTab *self)
{
  g_autofree char *default_container = NULL;
  g_autoptr(BevyIpcContainer) container = NULL;
  g_autoptr(VtePty) new_pty = NULL;
  BevyApplication *app;
  const char *profile_uuid;
  const char *cwd_uri;
  VtePty *pty;

  g_assert (BEVY_IS_TAB (self));
  g_assert (self->state == BEVY_TAB_STATE_INITIAL ||
            self->state == BEVY_TAB_STATE_EXITED ||
            self->state == BEVY_TAB_STATE_FAILED);

  gtk_widget_set_visible (GTK_WIDGET (self->banner), FALSE);

  app = BEVY_APPLICATION_DEFAULT;
  profile_uuid = bevy_profile_get_uuid (self->profile);
  default_container = bevy_profile_dup_default_container (self->profile);

  if (self->container_at_creation != NULL)
    container = g_object_ref (self->container_at_creation);
  else
    container = bevy_application_lookup_container (app, default_container);

  if (container == NULL)
    {
      g_autofree char *title = NULL;

      self->state = BEVY_TAB_STATE_FAILED;

      title = g_strdup_printf (_("Cannot locate container “%s”"), default_container);
      adw_banner_set_title (self->banner, title);
      adw_banner_set_button_label (self->banner, _("Edit Profile"));
      gtk_actionable_set_action_target (GTK_ACTIONABLE (self->banner), "s", profile_uuid);
      gtk_actionable_set_action_name (GTK_ACTIONABLE (self->banner), "app.edit-profile");
      gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);

      return;
    }

  self->state = BEVY_TAB_STATE_SPAWNING;

  pty = vte_terminal_get_pty (VTE_TERMINAL (self->terminal));

  if (pty == NULL)
    {
      g_autoptr(GError) error = NULL;

      new_pty = bevy_application_create_pty (BEVY_APPLICATION_DEFAULT, &error);

      if (new_pty == NULL)
        {
          self->state = BEVY_TAB_STATE_FAILED;

          adw_banner_set_title (self->banner, _("Failed to create pseudo terminal device"));
          adw_banner_set_button_label (self->banner, NULL);
          gtk_actionable_set_action_name (GTK_ACTIONABLE (self->banner), NULL);
          gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);

          return;
        }

      vte_terminal_set_pty (VTE_TERMINAL (self->terminal), new_pty);

      pty = new_pty;
    }

  cwd_uri = self->previous_working_directory_uri;
  if (self->initial_working_directory_uri)
    cwd_uri = self->initial_working_directory_uri;

  bevy_application_spawn_async (BEVY_APPLICATION_DEFAULT,
                                  container,
                                  self->profile,
                                  cwd_uri,
                                  pty,
                                  (const char * const *)self->command,
                                  NULL,
                                  bevy_tab_spawn_cb,
                                  g_object_ref (self));

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);
}

static void
bevy_tab_respawn_action (GtkWidget  *widget,
                           const char *action_name,
                           GVariant   *params)
{
  BevyTab *self = (BevyTab *)widget;

  g_assert (BEVY_IS_TAB (self));

  if (self->state == BEVY_TAB_STATE_FAILED ||
      self->state == BEVY_TAB_STATE_EXITED)
    bevy_tab_respawn (self);
}


static void
bevy_tab_inspect_action (GtkWidget  *widget,
                           const char *action_name,
                           GVariant   *params)
{
  BevyTab *self = (BevyTab *)widget;
  BevyInspector *inspector;
  GtkRoot *root;

  g_assert (BEVY_IS_TAB (self));

  inspector = bevy_inspector_new (self);
  root = gtk_widget_get_root (GTK_WIDGET (self));

  gtk_window_set_transient_for (GTK_WINDOW (inspector), GTK_WINDOW (root));
  gtk_window_set_modal (GTK_WINDOW (inspector), FALSE);
  gtk_window_present (GTK_WINDOW (inspector));
}

static void
bevy_tab_map (GtkWidget *widget)
{
  BevyTab *self = (BevyTab *)widget;

  g_assert (BEVY_IS_TAB (widget));

  GTK_WIDGET_CLASS (bevy_tab_parent_class)->map (widget);

  if (self->state == BEVY_TAB_STATE_INITIAL)
    bevy_tab_respawn (self);
}

static void
bevy_tab_notify_contains_focus_cb (BevyTab               *self,
                                     GParamSpec              *pspec,
                                     GtkEventControllerFocus *focus)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (GTK_IS_EVENT_CONTROLLER_FOCUS (focus));

  if (gtk_event_controller_focus_contains_focus (focus))
    {
      bevy_tab_set_needs_attention (self, FALSE);
      g_application_withdraw_notification (G_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                           self->uuid);
    }
}

static void
bevy_tab_notify_window_title_cb (BevyTab      *self,
                                   GParamSpec     *pspec,
                                   BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);
}

static void
bevy_tab_notify_window_subtitle_cb (BevyTab      *self,
                                      BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_SUBTITLE]);
}

static void
bevy_tab_increase_font_size_cb (BevyTab      *self,
                                  BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  bevy_tab_zoom_in (self);
}

static void
bevy_tab_decrease_font_size_cb (BevyTab      *self,
                                  BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  bevy_tab_zoom_out (self);
}

static void
bevy_tab_bell_cb (BevyTab      *self,
                    BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  g_signal_emit (self, signals[BELL], 0);
}

static BevyIpcContainer *
bevy_tab_discover_container (BevyTab *self)
{
  const char *current_container_name = bevy_terminal_get_current_container_name (self->terminal);
  const char *current_container_runtime = bevy_terminal_get_current_container_runtime (self->terminal);

  return bevy_application_find_container_by_name (BEVY_APPLICATION_DEFAULT,
                                                    current_container_runtime,
                                                    current_container_name);
}

static GIcon *
bevy_tab_dup_icon (BevyTab *self)
{
  BevyProcessLeaderKind kind;

  g_assert (BEVY_IS_TAB (self));

  kind = self->leader_kind;

  switch (kind)
    {
    default:
    case BEVY_PROCESS_LEADER_KIND_REMOTE:
      return g_themed_icon_new ("process-remote-symbolic");

    case BEVY_PROCESS_LEADER_KIND_SUPERUSER:
      return g_themed_icon_new ("process-superuser-symbolic");

    case BEVY_PROCESS_LEADER_KIND_CONTAINER:
    case BEVY_PROCESS_LEADER_KIND_UNKNOWN:
      {
        g_autoptr(BevyIpcContainer) container = NULL;
        const char *icon_name;

        if (!(container = bevy_tab_discover_container (self)))
          {
            if (!g_set_object (&container, self->container_at_creation))
              {
                if (self->profile != NULL)
                {
                  g_autofree char *profile_uuid = bevy_profile_dup_default_container (self->profile);

                  container = bevy_application_lookup_container (BEVY_APPLICATION_DEFAULT, profile_uuid);
                }
              }
          }

        if (container != NULL &&
            (icon_name = bevy_ipc_container_get_icon_name (container)) &&
            icon_name[0] != 0)
          return g_themed_icon_new (icon_name);
      }
      return NULL;
    }
}

static void
bevy_tab_invalidate_thumbnail (BevyTab *self)
{
  GtkWidget *view;
  AdwTabPage *page;

  g_assert (BEVY_IS_TAB (self));

  g_clear_object (&self->cached_texture);

  gtk_widget_queue_draw (GTK_WIDGET (self));

  if ((view = gtk_widget_get_ancestor (GTK_WIDGET (self), ADW_TYPE_TAB_VIEW)) &&
      (page = adw_tab_view_get_page (ADW_TAB_VIEW (view), GTK_WIDGET (self))))
    adw_tab_page_invalidate_thumbnail (page);
}

static void
bevy_tab_notify_palette_cb (BevyTab      *self,
                              GParamSpec     *pspec,
                              BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  bevy_tab_invalidate_thumbnail (self);
}

static void
bevy_tab_update_scrollbar_policy (BevyTab *self)
{
  BevySettings *settings;
  BevyScrollbarPolicy policy;

  g_assert (BEVY_IS_TAB (self));

  settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);
  policy = bevy_settings_get_scrollbar_policy (settings);

  switch (policy)
    {
    case BEVY_SCROLLBAR_POLICY_NEVER:
      gtk_scrolled_window_set_overlay_scrolling (self->scrolled_window, FALSE);
      gtk_scrolled_window_set_policy (self->scrolled_window, GTK_POLICY_NEVER, GTK_POLICY_EXTERNAL);
      break;

    case BEVY_SCROLLBAR_POLICY_ALWAYS:
      gtk_scrolled_window_set_overlay_scrolling (self->scrolled_window, FALSE);
      gtk_scrolled_window_set_policy (self->scrolled_window, GTK_POLICY_NEVER, GTK_POLICY_ALWAYS);
      break;

    case BEVY_SCROLLBAR_POLICY_SYSTEM:
      if (bevy_application_get_overlay_scrollbars (BEVY_APPLICATION_DEFAULT))
        {
          gtk_scrolled_window_set_overlay_scrolling (self->scrolled_window, TRUE);
          gtk_scrolled_window_set_policy (self->scrolled_window, GTK_POLICY_NEVER, GTK_POLICY_AUTOMATIC);
        }
      else
        {
          gtk_scrolled_window_set_overlay_scrolling (self->scrolled_window, FALSE);
          gtk_scrolled_window_set_policy (self->scrolled_window, GTK_POLICY_NEVER, GTK_POLICY_ALWAYS);
        }

      break;

    default:
      g_assert_not_reached ();
    }
}

static void
bevy_tab_update_word_char_exceptions (BevyTab      *self,
                                        GParamSpec     *pspec,
                                        BevySettings *settings)
{
  g_autofree char *word_char_exceptions = NULL;

  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_SETTINGS (settings));

  word_char_exceptions = bevy_settings_dup_word_char_exceptions (settings);
  vte_terminal_set_word_char_exceptions (VTE_TERMINAL (self->terminal), word_char_exceptions);
}

static void
bevy_tab_constructed (GObject *object)
{
  BevyTab *self = (BevyTab *)object;
  BevySettings *settings;
  g_autofree char *css_class = NULL;

  G_OBJECT_CLASS (bevy_tab_parent_class)->constructed (object);

  settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);
  g_object_bind_property (settings, "audible-bell",
                          self->terminal, "audible-bell",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "cursor-shape",
                          self->terminal, "cursor-shape",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "cursor-blink-mode",
                          self->terminal, "cursor-blink-mode",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "enable-a11y",
                          self->terminal, "enable-a11y",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "font-desc",
                          self->terminal, "font-desc",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "text-blink-mode",
                          self->terminal, "text-blink-mode",
                          G_BINDING_SYNC_CREATE);
  g_object_bind_property (settings, "ignore-osc-title",
                          self, "ignore-osc-title",
                          G_BINDING_SYNC_CREATE);

  css_class = g_strdup_printf ("bevy-tab-%s", self->uuid);
  gtk_widget_add_css_class (GTK_WIDGET (self->terminal), css_class);
  self->css_provider = gtk_css_provider_new ();
  if (gdk_display_get_default () != NULL)
    gtk_style_context_add_provider_for_display (gdk_display_get_default (),
                                                GTK_STYLE_PROVIDER (self->css_provider),
                                                GTK_STYLE_PROVIDER_PRIORITY_USER + 1);
  bevy_tab_update_margins (self);

  g_signal_connect_object (BEVY_APPLICATION_DEFAULT,
                           "notify::overlay-scrollbars",
                           G_CALLBACK (bevy_tab_update_scrollbar_policy),
                           self,
                           G_CONNECT_SWAPPED);
  g_signal_connect_object (settings,
                           "notify::scrollbar-policy",
                           G_CALLBACK (bevy_tab_update_scrollbar_policy),
                           self,
                           G_CONNECT_SWAPPED);
  bevy_tab_update_scrollbar_policy (self);

  /* Set up signal group for profile signals */
  self->profile_signals = g_signal_group_new (BEVY_TYPE_PROFILE);
  g_signal_connect_object (self->profile_signals,
                           "bind",
                           G_CALLBACK (bevy_tab_profile_signals_bind_cb),
                           self,
                           G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::limit-scrollback",
                                 G_CALLBACK (bevy_tab_update_scrollback_lines),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::scrollback-lines",
                                 G_CALLBACK (bevy_tab_update_scrollback_lines),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::cell-height-scale",
                                 G_CALLBACK (bevy_tab_update_cell_height_scale),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::cell-width-scale",
                                 G_CALLBACK (bevy_tab_update_cell_width_scale),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::margin-x",
                                 G_CALLBACK (bevy_tab_update_margins),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "notify::margin-y",
                                 G_CALLBACK (bevy_tab_update_margins),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_connect_object (self->profile_signals,
                                 "custom-links-changed",
                                 G_CALLBACK (bevy_tab_update_custom_links),
                                 self,
                                 G_CONNECT_SWAPPED);
  g_signal_group_set_target (self->profile_signals, self->profile);

  g_signal_connect_object (settings,
                           "notify::word-char-exceptions",
                           G_CALLBACK (bevy_tab_update_word_char_exceptions),
                           self,
                           G_CONNECT_SWAPPED);
  bevy_tab_update_word_char_exceptions (self, NULL, settings);

  g_signal_connect_object (settings,
                           "notify::inhibit-logout",
                           G_CALLBACK (bevy_tab_update_inhibit),
                           self,
                           G_CONNECT_SWAPPED);
  bevy_tab_update_inhibit (self);

  self->monitor = bevy_tab_monitor_new (self);
}

static void
bevy_tab_profile_signals_bind_cb (BevyTab     *self,
                                    BevyProfile *profile,
                                    GSignalGroup  *group)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_PROFILE (profile));
  g_assert (G_IS_SIGNAL_GROUP (group));

  /* Trigger all update functions when profile changes */
  bevy_tab_update_scrollback_lines (self);
  bevy_tab_update_cell_height_scale (self);
  bevy_tab_update_cell_width_scale (self);
  bevy_tab_update_margins (self);
  bevy_tab_update_custom_links (self);
}

static void
bevy_tab_snapshot (GtkWidget   *widget,
                     GtkSnapshot *snapshot)
{
  BevyTab *self = (BevyTab *)widget;
  BevyWindow *window;
  GdkRGBA bg;
  gboolean animating;
  int width;
  int height;

  g_assert (BEVY_IS_TAB (self));
  g_assert (GTK_IS_SNAPSHOT (snapshot));

  if (self->ignore_snapshot)
    return;

  window = BEVY_WINDOW (gtk_widget_get_root (widget));
  animating = bevy_window_is_animating (window);
  width = gtk_widget_get_width (widget);
  height = gtk_widget_get_height (widget);

  vte_terminal_get_color_background_for_draw (VTE_TERMINAL (self->terminal), &bg);

  if (animating &&
      bevy_window_get_active_tab (window) == self)
    {

      if (self->cached_texture == NULL)
        {
          GtkSnapshot *sub_snapshot = gtk_snapshot_new ();
          int scale_factor = gtk_widget_get_scale_factor (widget);
          g_autoptr(GskRenderNode) node = NULL;
          graphene_matrix_t matrix;
          GskRenderer *renderer;

          gtk_snapshot_scale (sub_snapshot, scale_factor, scale_factor);
          gtk_snapshot_append_color (sub_snapshot,
                                     &bg,
                                     &GRAPHENE_RECT_INIT (0, 0, width, height));

          if (gtk_widget_compute_transform (GTK_WIDGET (self->terminal),
                                            GTK_WIDGET (self),
                                            &matrix))
            {
              gtk_snapshot_transform_matrix (sub_snapshot, &matrix);
              GTK_WIDGET_GET_CLASS (self->terminal)->snapshot (GTK_WIDGET (self->terminal), sub_snapshot);
            }

          node = gtk_snapshot_free_to_node (sub_snapshot);
          renderer = gtk_native_get_renderer (GTK_NATIVE (window));

          self->cached_texture = gsk_renderer_render_texture (renderer,
                                                              node,
                                                              &GRAPHENE_RECT_INIT (0,
                                                                                   0,
                                                                                   width * scale_factor,
                                                                                   height * scale_factor));
        }

      gtk_snapshot_append_texture (snapshot,
                                   self->cached_texture,
                                   &GRAPHENE_RECT_INIT (0, 0, width, height));
    }
  else
    {
      g_clear_object (&self->cached_texture);

      if (animating)
        gtk_snapshot_append_color (snapshot,
                                   &bg,
                                   &GRAPHENE_RECT_INIT (0, 0, width, height));

      GTK_WIDGET_CLASS (bevy_tab_parent_class)->snapshot (widget, snapshot);
    }
}

static void
bevy_tab_size_allocate (GtkWidget *widget,
                          int        width,
                          int        height,
                          int        baseline)
{
  BevyTab *self = (BevyTab *)widget;

  g_assert (BEVY_IS_TAB (self));

  GTK_WIDGET_CLASS (bevy_tab_parent_class)->size_allocate (widget, width, height, baseline);

  g_clear_object (&self->cached_texture);
}

static void
bevy_tab_invalidate_icon (BevyTab *self)
{
  g_assert (BEVY_IS_TAB (self));

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ICON]);
}

static void
bevy_tab_invalidate_progress (BevyTab *self)
{
  g_assert (BEVY_IS_TAB (self));

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROGRESS]);
  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROGRESS_FRACTION]);
  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_INDICATOR_ICON]);
}

static gboolean
bevy_tab_match_clicked_cb (BevyTab       *self,
                             double           x,
                             double           y,
                             int              button,
                             GdkModifierType  state,
                             const char      *match,
                             BevyTerminal  *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (match != NULL);
  g_assert (BEVY_IS_TERMINAL (terminal));

  if (!bevy_str_empty0 (match))
    {
      bevy_tab_open_uri (self, match);
      return TRUE;
    }

  return FALSE;
}

static void
bevy_tab_root (GtkWidget *widget)
{
  BevyTab *self = BEVY_TAB (widget);

  /* Clear our ignore_snapshot bit in case we've had our tab restored
   * from the parking lot.
   */
  self->ignore_snapshot = FALSE;

  GTK_WIDGET_CLASS (bevy_tab_parent_class)->root (widget);
}

static void
bevy_tab_unroot (GtkWidget *widget)
{
  BevyTab *self = BEVY_TAB (widget);

  /* Clear inhibit cookie when widget is unrooted since the window
   * reference may no longer be valid.
   */
  if (self->inhibit_cookie != 0)
    {
      gtk_application_uninhibit (GTK_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                 self->inhibit_cookie);
      self->inhibit_cookie = 0;
    }

  GTK_WIDGET_CLASS (bevy_tab_parent_class)->unroot (widget);
}

static void
bevy_tab_commit_cb (BevyTab      *self,
                      const char     *str,
                      guint           length,
                      BevyTerminal *terminal)
{
  g_assert (BEVY_IS_TAB (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  g_signal_emit (self, signals[COMMIT], 0, str);
}

static void
bevy_tab_dispose (GObject *object)
{
  BevyTab *self = (BevyTab *)object;
  GtkWidget *child;

  g_debug ("Disposing tab");

  bevy_tab_notify_destroy (&self->notify);

  bevy_tab_force_quit (self);

  gtk_widget_dispose_template (GTK_WIDGET (self), BEVY_TYPE_TAB);

  while ((child = gtk_widget_get_first_child (GTK_WIDGET (self))))
    gtk_widget_unparent (child);

  g_clear_object (&self->cached_texture);
  g_clear_object (&self->css_provider);
  g_clear_object (&self->profile);
  g_clear_object (&self->profile_signals);
  g_clear_object (&self->process);
  g_clear_object (&self->monitor);
  g_clear_object (&self->container_at_creation);

  if (self->inhibit_cookie != 0)
    {
      gtk_application_uninhibit (GTK_APPLICATION (BEVY_APPLICATION_DEFAULT),
                                 self->inhibit_cookie);
      self->inhibit_cookie = 0;
    }

  g_clear_pointer (&self->initial_working_directory_uri, g_free);
  g_clear_pointer (&self->previous_working_directory_uri, g_free);
  g_clear_pointer (&self->title_prefix, g_free);
  g_clear_pointer (&self->initial_title, g_free);
  g_clear_pointer (&self->command, g_strfreev);
  g_clear_pointer (&self->command_line, g_free);
  g_clear_pointer (&self->program_name, g_free);

  G_OBJECT_CLASS (bevy_tab_parent_class)->dispose (object);
}

static void
bevy_tab_finalize (GObject *object)
{
  BevyTab *self = (BevyTab *)object;

  g_clear_pointer (&self->uuid, g_free);

  G_OBJECT_CLASS (bevy_tab_parent_class)->finalize (object);
}

static void
bevy_tab_get_property (GObject    *object,
                         guint       prop_id,
                         GValue     *value,
                         GParamSpec *pspec)
{
  BevyTab *self = BEVY_TAB (object);

  switch (prop_id)
    {
    case PROP_COMMAND_LINE:
      g_value_set_string (value, self->command_line);
      break;

    case PROP_ICON:
      g_value_take_object (value, bevy_tab_dup_icon (self));
      break;

    case PROP_IGNORE_OSC_TITLE:
      g_value_set_boolean (value, bevy_tab_get_ignore_osc_title (self));
      break;

    case PROP_INDICATOR_ICON:
      g_value_take_object (value, bevy_tab_dup_indicator_icon (self));
      break;

    case PROP_PROCESS_LEADER_KIND:
      g_value_set_enum (value, self->leader_kind);
      break;

    case PROP_PROGRESS:
      g_value_set_enum (value, bevy_tab_get_progress (self));
      break;

    case PROP_PROGRESS_FRACTION:
      g_value_set_double (value, bevy_tab_get_progress_fraction (self));
      break;

    case PROP_PROFILE:
      g_value_set_object (value, bevy_tab_get_profile (self));
      break;

    case PROP_READ_ONLY:
      g_value_set_boolean (value, !vte_terminal_get_input_enabled (VTE_TERMINAL (self->terminal)));
      break;

    case PROP_SUBTITLE:
      g_value_take_string (value, bevy_tab_dup_subtitle (self));
      break;

    case PROP_TITLE:
      g_value_take_string (value, bevy_tab_dup_title (self));
      break;

    case PROP_TITLE_PREFIX:
      g_value_set_string (value, bevy_tab_get_title_prefix (self));
      break;

    case PROP_UUID:
      g_value_set_string (value, bevy_tab_get_uuid (self));
      break;

    case PROP_ZOOM:
      g_value_set_enum (value, bevy_tab_get_zoom (self));
      break;

    case PROP_ZOOM_LABEL:
      g_value_take_string (value, bevy_tab_dup_zoom_label (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_tab_set_property (GObject      *object,
                         guint         prop_id,
                         const GValue *value,
                         GParamSpec   *pspec)
{
  BevyTab *self = BEVY_TAB (object);

  switch (prop_id)
    {
    case PROP_IGNORE_OSC_TITLE:
      bevy_tab_set_ignore_osc_title (self, g_value_get_boolean (value));
      break;

    case PROP_PROFILE:
      self->profile = g_value_dup_object (value);
      break;

    case PROP_READ_ONLY:
      vte_terminal_set_input_enabled (VTE_TERMINAL (self->terminal), !g_value_get_boolean (value));
      break;

    case PROP_TITLE_PREFIX:
      bevy_tab_set_title_prefix (self, g_value_get_string (value));
      break;

    case PROP_ZOOM:
      bevy_tab_set_zoom (self, g_value_get_enum (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_tab_class_init (BevyTabClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->constructed = bevy_tab_constructed;
  object_class->dispose = bevy_tab_dispose;
  object_class->finalize = bevy_tab_finalize;
  object_class->get_property = bevy_tab_get_property;
  object_class->set_property = bevy_tab_set_property;

  widget_class->map = bevy_tab_map;
  widget_class->snapshot = bevy_tab_snapshot;
  widget_class->size_allocate = bevy_tab_size_allocate;
  widget_class->root = bevy_tab_root;
  widget_class->unroot = bevy_tab_unroot;

  properties[PROP_COMMAND_LINE] =
    g_param_spec_string ("command-line", NULL, NULL,
                         NULL,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_ICON] =
    g_param_spec_object ("icon", NULL, NULL,
                         G_TYPE_ICON,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_IGNORE_OSC_TITLE] =
    g_param_spec_boolean ("ignore-osc-title", NULL, NULL,
                         FALSE,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_INDICATOR_ICON] =
    g_param_spec_object ("indicator-icon", NULL, NULL,
                         G_TYPE_ICON,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_PROCESS_LEADER_KIND] =
    g_param_spec_enum ("process-leader-kind", NULL, NULL,
                       BEVY_TYPE_PROCESS_LEADER_KIND,
                       BEVY_PROCESS_LEADER_KIND_UNKNOWN,
                       (G_PARAM_READABLE |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_PROFILE] =
    g_param_spec_object ("profile", NULL, NULL,
                         BEVY_TYPE_PROFILE,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_PROGRESS] =
    g_param_spec_enum ("progress", NULL, NULL,
                       BEVY_TYPE_TAB_PROGRESS,
                       BEVY_TAB_PROGRESS_INDETERMINATE,
                       (G_PARAM_READABLE |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_PROGRESS_FRACTION] =
    g_param_spec_double ("progress-fraction", NULL, NULL,
                         0, 1, 0,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_READ_ONLY] =
    g_param_spec_boolean ("read-only", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_SUBTITLE] =
    g_param_spec_string ("subtitle", NULL, NULL,
                         NULL,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_TITLE] =
    g_param_spec_string ("title", NULL, NULL,
                         NULL,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_TITLE_PREFIX] =
    g_param_spec_string ("title-prefix", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_UUID] =
    g_param_spec_string ("uuid", NULL, NULL,
                         NULL,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_ZOOM] =
    g_param_spec_enum ("zoom", NULL, NULL,
                       BEVY_TYPE_ZOOM_LEVEL,
                       BEVY_ZOOM_LEVEL_DEFAULT,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_ZOOM_LABEL] =
    g_param_spec_string ("zoom-label", NULL, NULL,
                         NULL,
                         (G_PARAM_READABLE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  signals[BELL] =
    g_signal_new_class_handler ("bell",
                                G_TYPE_FROM_CLASS (klass),
                                G_SIGNAL_RUN_LAST,
                                NULL,
                                NULL, NULL,
                                NULL,
                                G_TYPE_NONE, 0);

  signals[COMMIT] =
    g_signal_new_class_handler ("commit",
                                G_TYPE_FROM_CLASS (klass),
                                G_SIGNAL_RUN_LAST,
                                NULL,
                                NULL, NULL,
                                NULL,
                                G_TYPE_NONE,
                                1,
                                G_TYPE_STRING | G_SIGNAL_TYPE_STATIC_SCOPE);

  gtk_widget_class_set_template_from_resource (widget_class, "/dev/itznoel/Bevy/bevy-tab.ui");
  gtk_widget_class_set_layout_manager_type (widget_class, GTK_TYPE_BIN_LAYOUT);
  gtk_widget_class_set_css_name (widget_class, "bevytab");

  gtk_widget_class_bind_template_child (widget_class, BevyTab, banner);
  gtk_widget_class_bind_template_child (widget_class, BevyTab, terminal);
  gtk_widget_class_bind_template_child (widget_class, BevyTab, scrolled_window);

  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_notify_contains_focus_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_notify_window_title_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_notify_window_subtitle_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_increase_font_size_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_decrease_font_size_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_notify_palette_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_bell_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_invalidate_icon);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_invalidate_progress);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_match_clicked_cb);
  gtk_widget_class_bind_template_callback (widget_class, bevy_tab_commit_cb);

  gtk_widget_class_install_action (widget_class, "tab.respawn", NULL, bevy_tab_respawn_action);
  gtk_widget_class_install_action (widget_class, "tab.inspect", NULL, bevy_tab_inspect_action);

  g_type_ensure (BEVY_TYPE_TERMINAL);
}

static void
bevy_tab_init (BevyTab *self)
{
  GtkEventController *controller;

  self->state = BEVY_TAB_STATE_INITIAL;
  self->zoom = BEVY_ZOOM_LEVEL_DEFAULT;
  self->uuid = g_uuid_string_random ();

  gtk_widget_init_template (GTK_WIDGET (self));

  bevy_tab_notify_init (&self->notify, self);

  controller = gtk_event_controller_scroll_new (GTK_EVENT_CONTROLLER_SCROLL_VERTICAL);
  gtk_event_controller_set_propagation_phase (controller, GTK_PHASE_CAPTURE);
  g_signal_connect (controller,
                    "scroll",
                    G_CALLBACK (on_scroll_scrolled_cb),
                    self);
  g_signal_connect (controller,
                    "scroll-begin",
                    G_CALLBACK (on_scroll_begin_cb),
                    self);
  g_signal_connect (controller,
                    "scroll-end",
                    G_CALLBACK (on_scroll_end_cb),
                    self);
  gtk_widget_add_controller (GTK_WIDGET (self), controller);

  /* Ensure we redraw when the dark-mode changes so that if the user
   * goes to the tab-overview all the tabs look correct.
   */
  g_signal_connect_object (adw_style_manager_get_default (),
                           "notify::dark",
                           G_CALLBACK (bevy_tab_invalidate_thumbnail),
                           self,
                           G_CONNECT_SWAPPED);
}

BevyTab *
bevy_tab_new (BevyProfile *profile)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (profile), NULL);

  return g_object_new (BEVY_TYPE_TAB,
                       "profile", profile,
                       NULL);
}

/**
 * bevy_tab_get_profile:
 * @self: a #BevyTab
 *
 * Gets the profile used by the tab.
 *
 * Returns: (transfer none) (not nullable): a #BevyProfile
 */
BevyProfile *
bevy_tab_get_profile (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->profile;
}

/**
 * bevy_tab_apply_profile:
 * @self: a #BevyTab
 * @new_profile: a #BevyProfile to apply
 *
 * Applies a profile to the tab by replacing the tab's profile reference
 * with @new_profile. The tab will share the profile with other tabs,
 * so when the profile is edited in preferences, all tabs using it will
 * be updated automatically.
 */
void
bevy_tab_apply_profile (BevyTab     *self,
                          BevyProfile *new_profile)
{
  g_return_if_fail (BEVY_IS_TAB (self));
  g_return_if_fail (BEVY_IS_PROFILE (new_profile));

  /* Don't do anything if it's already the same profile */
  if (self->profile == new_profile)
    return;

  /* Replace the profile with the selected one. */
  g_clear_object (&self->profile);
  self->profile = g_object_ref (new_profile);


  g_signal_group_set_target (self->profile_signals, self->profile);

  /* Notify that the profile property changed */
  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROFILE]);
}

const char *
bevy_tab_get_title_prefix (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->title_prefix ? self->title_prefix : "";
}

void
bevy_tab_set_title_prefix (BevyTab  *self,
                             const char *title_prefix)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  if (bevy_str_empty0 (title_prefix))
    title_prefix = NULL;

  if (g_set_str (&self->title_prefix, title_prefix))
    {
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE_PREFIX]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);
    }
}

char *
bevy_tab_dup_title (BevyTab *self)
{
  GString *gstr;

  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  gstr = g_string_new (self->title_prefix);

  if (!self->ignore_osc_title)
    {
      const char *window_title;

      G_GNUC_BEGIN_IGNORE_DEPRECATIONS
        window_title = vte_terminal_get_window_title (VTE_TERMINAL (self->terminal));
      G_GNUC_END_IGNORE_DEPRECATIONS

      if (window_title && window_title[0])
        g_string_append (gstr, window_title);
      else if (self->command != NULL && self->command[0] != NULL)
        g_string_append (gstr, self->command[0]);
      else if (self->initial_title != NULL)
        g_string_append (gstr, self->initial_title);
    }

  if (gstr->len == 0)
    g_string_append (gstr, _("Terminal"));

  if (self->state == BEVY_TAB_STATE_EXITED)
    g_string_append_printf (gstr, " (%s)", _("Exited"));
  else if (self->state == BEVY_TAB_STATE_FAILED)
    g_string_append_printf (gstr, " (%s)", _("Failed"));
  else if (self->has_foreground_process &&
           !bevy_str_empty0 (self->command_line) &&
           !bevy_str_empty0 (self->program_name) &&
           !bevy_is_shell (self->program_name))
    g_string_append_printf (gstr, " — %s", self->command_line);

  return g_string_free (gstr, FALSE);
}

static char *
bevy_tab_collapse_uri (const char *uri)
{
  g_autoptr(GFile) file = NULL;

  if (uri == NULL)
    return NULL;

  if (!(file = g_file_new_for_uri (uri)))
    return NULL;

  if (g_file_is_native (file))
    return bevy_path_collapse (g_file_peek_path (file));

  return strdup (uri);
}

char *
bevy_tab_dup_subtitle (BevyTab *self)
{
  g_autofree char *current_directory_uri = NULL;
  g_autofree char *current_file_uri = NULL;

  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  current_file_uri = bevy_terminal_dup_current_file_uri (self->terminal);
  if (current_file_uri != NULL && current_file_uri[0] != 0)
    return bevy_tab_collapse_uri (current_file_uri);

  current_directory_uri = bevy_terminal_dup_current_directory_uri (self->terminal);
  if (current_directory_uri != NULL && current_directory_uri[0] != 0)
    return bevy_tab_collapse_uri (current_directory_uri);

  return g_strdup ("");
}

char *
bevy_tab_dup_current_directory_uri (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return bevy_terminal_dup_current_directory_uri (self->terminal);
}

void
bevy_tab_set_initial_working_directory_uri (BevyTab  *self,
                                              const char *initial_working_directory_uri)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  g_set_str (&self->initial_working_directory_uri, initial_working_directory_uri);
}

char *
bevy_tab_dup_previous_working_directory_uri (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return g_strdup (self->previous_working_directory_uri);
}


void
bevy_tab_set_previous_working_directory_uri (BevyTab  *self,
                                               const char *previous_working_directory_uri)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  g_set_str (&self->previous_working_directory_uri, previous_working_directory_uri);
}

static void
bevy_tab_apply_zoom (BevyTab *self)
{
  g_assert (BEVY_IS_TAB (self));

  vte_terminal_set_font_scale (VTE_TERMINAL (self->terminal),
                               zoom_font_scales[self->zoom]);
}

BevyZoomLevel
bevy_tab_get_zoom (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), 0);

  return self->zoom;
}

void
bevy_tab_set_zoom (BevyTab       *self,
                     BevyZoomLevel  zoom)
{
  g_return_if_fail (BEVY_IS_TAB (self));
  g_return_if_fail (zoom >= BEVY_ZOOM_LEVEL_MINUS_14 &&
                    zoom <= BEVY_ZOOM_LEVEL_PLUS_14);

  if (zoom != self->zoom)
    {
      self->zoom = zoom;
      bevy_tab_apply_zoom (self);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ZOOM]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ZOOM_LABEL]);
    }
}

void
bevy_tab_zoom_in (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  if (self->zoom < BEVY_ZOOM_LEVEL_PLUS_14)
    bevy_tab_set_zoom (self, self->zoom + 1);
}

void
bevy_tab_zoom_out (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  if (self->zoom > BEVY_ZOOM_LEVEL_MINUS_14)
    bevy_tab_set_zoom (self, self->zoom - 1);
}

BevyTerminal *
bevy_tab_get_terminal (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->terminal;
}

void
bevy_tab_raise (BevyTab *self)
{
  AdwTabView *tab_view;
  AdwTabPage *tab_page;

  g_return_if_fail (BEVY_IS_TAB (self));

  if ((tab_view = ADW_TAB_VIEW (gtk_widget_get_ancestor (GTK_WIDGET (self), ADW_TYPE_TAB_VIEW))) &&
      (tab_page = adw_tab_view_get_page (tab_view, GTK_WIDGET (self))))
    adw_tab_view_set_selected_page (tab_view, tab_page);
}

typedef struct _Wait
{
  GMainContext *context;
  gboolean completed;
  gboolean success;
} Wait;

static void
bevy_tab_poll_agent_sync_cb (GObject      *object,
                               GAsyncResult *result,
                               gpointer      user_data)
{
  BevyTab *self = (BevyTab *)object;
  Wait *wait = user_data;

  g_assert (BEVY_IS_TAB (self));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (wait != NULL);

  wait->completed = TRUE;
  wait->success = bevy_tab_poll_agent_finish (self, result, NULL);

  g_main_context_wakeup (wait->context);
}

static gboolean
bevy_tab_poll_agent (BevyTab *self)
{
  Wait wait;

  g_return_val_if_fail (BEVY_IS_TAB (self), FALSE);

  wait.context = g_main_context_get_thread_default ();
  wait.completed = FALSE;
  wait.success = FALSE;

  bevy_tab_poll_agent_async (self,
                               NULL,
                               bevy_tab_poll_agent_sync_cb,
                               &wait);

  while (!wait.completed)
    g_main_context_iteration (wait.context, TRUE);

  return wait.success;
}

/**
 * bevy_tab_is_running:
 * @self: a #BevyTab
 * @cmdline: (out) (nullable): a location for the command line
 *
 * Returns: %TRUE if there is a command running
 */
gboolean
bevy_tab_is_running (BevyTab  *self,
                       char      **cmdline)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), FALSE);

  bevy_tab_poll_agent (self);

  if (cmdline != NULL)
    *cmdline = g_strdup (self->command_line);

  if (self->has_foreground_process && self->program_name != NULL)
    return !bevy_is_shell (self->program_name);

  return FALSE;
}

static gboolean
bevy_tab_force_quit_in_idle (gpointer data)
{
  BevyTab *self = data;

  g_assert (BEVY_IS_TAB (self));

  if (self->process != NULL)
    bevy_tab_send_signal (self, SIGKILL);

  return G_SOURCE_REMOVE;
}

void
bevy_tab_force_quit (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  g_debug ("Forcing tab to quit");

  self->forced_exit = TRUE;

  if (self->process == NULL)
    return;

  /* First we try to send SIGHUP so that shells like bash will save their
   * history (See #308).
   */
  bevy_tab_send_signal (self, SIGHUP);

  /* In case this was not enough for the process to actually exit, we setup
   * a short timer to send SIGKILL afterwards.
   */
  g_timeout_add_full (G_PRIORITY_HIGH,
                      50,
                      bevy_tab_force_quit_in_idle,
                      g_object_ref (self),
                      g_object_unref);
}

BevyIpcProcess *
bevy_tab_get_process (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->process;
}

char *
bevy_tab_dup_zoom_label (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), 0);

  if (self->zoom == BEVY_ZOOM_LEVEL_DEFAULT)
    return g_strdup ("100%");

  return g_strdup_printf ("%.0lf%%", zoom_font_scales[self->zoom] * 100.0);
}

void
bevy_tab_show_banner (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  gtk_widget_set_visible (GTK_WIDGET (self->banner), TRUE);
}

void
bevy_tab_set_needs_attention (BevyTab *self,
                                gboolean   needs_attention)
{
  GtkWidget *tab_view;
  AdwTabPage *page;

  g_return_if_fail (BEVY_IS_TAB (self));

  if ((tab_view = gtk_widget_get_ancestor (GTK_WIDGET (self), ADW_TYPE_TAB_VIEW)) &&
      (page = adw_tab_view_get_page (ADW_TAB_VIEW (tab_view), GTK_WIDGET (self))))
    adw_tab_page_set_needs_attention (page, needs_attention);
}

const char *
bevy_tab_get_uuid (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->uuid;
}

BevyIpcContainer *
bevy_tab_dup_container (BevyTab *self)
{
  g_autoptr(BevyIpcContainer) container = NULL;
  const char *runtime;
  const char *name;

  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  if ((runtime = bevy_terminal_get_current_container_runtime (self->terminal)) &&
      (name = bevy_terminal_get_current_container_name (self->terminal)))
    container = bevy_application_find_container_by_name (BEVY_APPLICATION_DEFAULT, runtime, name);

  if (container == NULL)
    g_set_object (&container, self->container_at_creation);

  return g_steal_pointer (&container);
}

void
bevy_tab_set_container (BevyTab          *self,
                          BevyIpcContainer *container)
{
  g_return_if_fail (BEVY_IS_TAB (self));
  g_return_if_fail (!container || BEVY_IPC_IS_CONTAINER (container));

  g_set_object (&self->container_at_creation, container);
}

static void
bevy_tab_poll_agent_cb (GObject      *object,
                          GAsyncResult *result,
                          gpointer      user_data)
{
  BevyIpcProcess *process = (BevyIpcProcess *)object;
  g_autoptr(GTask) task = user_data;
  g_autofree char *the_cmdline = NULL;
  g_autofree char *the_leader_kind = NULL;
  BevyProcessLeaderKind leader_kind;
  gboolean has_foreground_process;
  gboolean changed = FALSE;
  gboolean inhibit_changed = FALSE;
  BevyTab *self;
  GPid the_pid;

  g_assert (BEVY_IPC_IS_PROCESS (process));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (G_IS_TASK (task));

  self = g_task_get_source_object (task);

  g_assert (BEVY_IS_TAB (self));

  bevy_ipc_process_call_has_foreground_process_finish (process,
                                                         &has_foreground_process,
                                                         &the_pid,
                                                         &the_cmdline,
                                                         &the_leader_kind,
                                                         NULL,
                                                         result,
                                                         NULL);

  if (self->pid != the_pid)
    {
      changed = TRUE;
      self->pid = the_pid;
    }

  if (self->has_foreground_process != has_foreground_process)
    {
      changed = TRUE;
      inhibit_changed = TRUE;
      self->has_foreground_process = has_foreground_process;
    }

  if (g_strcmp0 (the_leader_kind, "superuser") == 0)
    leader_kind = BEVY_PROCESS_LEADER_KIND_SUPERUSER;
  else if (g_strcmp0 (the_leader_kind, "container") == 0)
    leader_kind = BEVY_PROCESS_LEADER_KIND_CONTAINER;
  else if (g_strcmp0 (the_leader_kind, "remote") == 0)
    leader_kind = BEVY_PROCESS_LEADER_KIND_REMOTE;
  else
    leader_kind = BEVY_PROCESS_LEADER_KIND_UNKNOWN;

  if (self->leader_kind != leader_kind)
    {
      changed = TRUE;
      self->leader_kind = leader_kind;

      if (!bevy_tab_is_active (self))
        bevy_tab_set_needs_attention (self, TRUE);

      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROCESS_LEADER_KIND]);
    }

  if (g_set_str (&self->command_line, the_cmdline))
    {
      g_autofree char *program_name = NULL;
      const char *space;

      changed = TRUE;

      if (the_cmdline != NULL && (space = strchr (the_cmdline, ' ')))
        program_name = g_strndup (the_cmdline, space - the_cmdline);

      if (g_set_str (&self->program_name, program_name))
        inhibit_changed = TRUE;

      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_COMMAND_LINE]);
    }

  if (changed)
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);

  if (inhibit_changed)
    bevy_tab_update_inhibit (self);

  g_task_return_boolean (task, changed);
}

void
bevy_tab_poll_agent_async (BevyTab           *self,
                             GCancellable        *cancellable,
                             GAsyncReadyCallback  callback,
                             gpointer             user_data)
{
  g_autoptr(GUnixFDList) fd_list = NULL;
  g_autoptr(GTask) task = NULL;
  VtePty *pty;
  int handle;
  int pty_fd;

  g_assert (BEVY_IS_TAB (self));

  task = g_task_new (self, cancellable, callback, user_data);
  g_task_set_source_tag (task, bevy_tab_poll_agent_async);

  if (self->process == NULL)
    {
      self->has_foreground_process = FALSE;
      self->pid = -1;

      if (g_set_str (&self->command_line, NULL))
        g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_COMMAND_LINE]);

      if (self->leader_kind != BEVY_PROCESS_LEADER_KIND_UNKNOWN)
        {
          self->leader_kind = BEVY_PROCESS_LEADER_KIND_UNKNOWN;
          g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROCESS_LEADER_KIND]);
        }

      g_task_return_boolean (task, FALSE);

      return;
    }

  pty = vte_terminal_get_pty (VTE_TERMINAL (self->terminal));
  pty_fd = vte_pty_get_fd (pty);
  fd_list = g_unix_fd_list_new ();
  handle = g_unix_fd_list_append (fd_list, pty_fd, NULL);

  bevy_ipc_process_call_has_foreground_process (self->process,
                                                  g_variant_new_handle (handle),
                                                  fd_list,
                                                  cancellable,
                                                  bevy_tab_poll_agent_cb,
                                                  g_steal_pointer (&task));


}

gboolean
bevy_tab_poll_agent_finish (BevyTab     *self,
                              GAsyncResult  *result,
                              GError       **error)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), FALSE);
  g_return_val_if_fail (G_IS_TASK (result), FALSE);

  return g_task_propagate_boolean (G_TASK (result), error);
}

gboolean
bevy_tab_has_foreground_process (BevyTab  *self,
                                   GPid       *pid,
                                   char      **cmdline)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), FALSE);

  bevy_tab_poll_agent (self);

  if (pid != NULL)
    *pid = self->pid;

  if (cmdline != NULL)
    *cmdline = g_strdup (self->command_line);

  return self->has_foreground_process;
}

void
bevy_tab_set_command (BevyTab          *self,
                        const char * const *command)
{
  char **copy;

  g_return_if_fail (BEVY_IS_TAB (self));

  if (command != NULL && command[0] == NULL)
    command = NULL;

  copy = g_strdupv ((char **)command);
  g_strfreev (self->command);
  self->command = copy;
}

const char *
bevy_tab_get_initial_title (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->initial_title;
}

void
bevy_tab_set_initial_title (BevyTab  *self,
                              const char *initial_title)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  g_set_str (&self->initial_title, initial_title);
}

const char *
bevy_tab_get_command_line (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  return self->command_line;
}

#ifdef __linux__
static void
bevy_tab_toast (BevyTab  *self,
                  int         timeout,
                  const char *title)
{
  GtkWidget *overlay = gtk_widget_get_ancestor (GTK_WIDGET (self), ADW_TYPE_TOAST_OVERLAY);
  AdwToast *toast;

  if (overlay == NULL)
    return;

  toast = g_object_new (ADW_TYPE_TOAST,
                        "title", title,
                        "timeout", timeout,
                        NULL);
  adw_toast_overlay_add_toast (ADW_TOAST_OVERLAY (overlay), toast);
}

static void
bevy_tab_open_uri_cb (GObject      *object,
                        GAsyncResult *result,
                        gpointer      user_data)
{
  g_autoptr(BevyTab) self = user_data;
  g_autoptr(GError) error = NULL;

  g_assert (XDP_IS_PORTAL (object));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (BEVY_IS_TAB (self));

  if (!xdp_portal_open_uri_finish (XDP_PORTAL (object), result, &error) &&
      !g_error_matches (error, G_IO_ERROR, G_IO_ERROR_CANCELLED))
    bevy_tab_toast (self, 3, _("Failed to open link"));
}

void
bevy_tab_open_uri (BevyTab  *self,
                     const char *uri)
{
  g_autofree char *translated = NULL;
  GtkWindow *window;
  XdpParent *parent;

  g_return_if_fail (BEVY_IS_TAB (self));
  g_return_if_fail (uri != NULL);

  window = GTK_WINDOW (gtk_widget_get_root (GTK_WIDGET (self)));

  if (g_str_has_prefix (uri, "file://"))
    {
      g_autoptr(BevyIpcContainer) container = bevy_tab_dup_container (self);
      g_autoptr(GUri) guri = NULL;

      if (container == NULL)
        {
          g_autofree char *default_container = bevy_profile_dup_default_container (self->profile);
          container = bevy_application_lookup_container (BEVY_APPLICATION_DEFAULT, default_container);
        }

      if (container != NULL)
        {
          if (bevy_ipc_container_call_translate_uri_sync (container, uri, &translated, NULL, NULL))
            uri = translated;
        }

      if (bevy_get_process_kind () == BEVY_PROCESS_KIND_FLATPAK &&
          (guri = g_uri_parse (uri, 0, NULL)) &&
          !g_str_has_prefix (g_uri_get_path (guri), g_get_home_dir ()))
        {
          const char *path = g_uri_get_path (guri);
          g_autofree char *new_path = g_build_filename ("/var/run/host", path, NULL);
          g_autoptr(GUri) rewritten = NULL;

          rewritten = g_uri_build (0,
                                   "file",
                                   g_uri_get_userinfo (guri),
                                   g_uri_get_host (guri),
                                   g_uri_get_port (guri),
                                   new_path,
                                   g_uri_get_query (guri),
                                   g_uri_get_fragment (guri));

          g_clear_pointer (&translated, g_free);
          uri = translated = g_uri_to_string (rewritten);
        }
    }
  else if (!g_utf8_strchr (uri, -1, ':') && g_utf8_strchr (uri, -1, '@'))
    {
      uri = translated = g_strconcat ("mailto:", uri, NULL);
    }

  if (portal == NULL)
    portal = xdp_portal_new ();

  parent = xdp_parent_new_gtk (window);
  xdp_portal_open_uri (portal,
                       parent,
                       uri,
                       XDP_OPEN_URI_FLAG_NONE,
                       NULL,
                       bevy_tab_open_uri_cb,
                       g_object_ref (self));
  xdp_parent_free (parent);
}
#else
void
bevy_tab_open_uri (BevyTab  *self,
                     const char *uri)
{
  G_GNUC_BEGIN_IGNORE_DEPRECATIONS
  gtk_show_uri (GTK_WINDOW (gtk_widget_get_root (GTK_WIDGET (self))), uri, 0);
  G_GNUC_END_IGNORE_DEPRECATIONS
}
#endif

char *
bevy_tab_query_working_directory_from_agent (BevyTab *self)
{
  g_autofree char *path = NULL;
  g_autoptr(GUnixFDList) fd_list = NULL;
  VtePty *pty;
  int pty_fd;
  int handle;

  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  if (self->process == NULL)
    return NULL;

  pty = vte_terminal_get_pty (VTE_TERMINAL (self->terminal));
  pty_fd = vte_pty_get_fd (pty);
  fd_list = g_unix_fd_list_new ();
  handle = g_unix_fd_list_append (fd_list, pty_fd, NULL);

  if (bevy_ipc_process_call_get_working_directory_sync (self->process,
                                                          g_variant_new_handle (handle),
                                                          fd_list,
                                                          &path,
                                                          NULL, NULL, NULL))
    return g_steal_pointer (&path);

  return NULL;
}

BevyTabProgress
bevy_tab_get_progress (BevyTab *self)
{
  gint64 state;

  g_return_val_if_fail (BEVY_IS_TAB (self), 0);

  if (vte_terminal_get_termprop_int_by_id (VTE_TERMINAL (self->terminal),
                                           VTE_PROPERTY_ID_PROGRESS_HINT,
                                           &state))
    {
      switch (state)
        {
        case VTE_PROGRESS_HINT_ACTIVE:
          return BEVY_TAB_PROGRESS_ACTIVE;

        case VTE_PROGRESS_HINT_ERROR:
          return BEVY_TAB_PROGRESS_ERROR;

        case VTE_PROGRESS_HINT_PAUSED:
        case VTE_PROGRESS_HINT_INDETERMINATE:
        default:
          return BEVY_TAB_PROGRESS_INDETERMINATE;
        }
    }

  return BEVY_TAB_PROGRESS_INDETERMINATE;
}

double
bevy_tab_get_progress_fraction (BevyTab *self)
{
  guint64 value;

  g_return_val_if_fail (BEVY_IS_TAB (self), .0);

  if (bevy_tab_get_progress (self) != BEVY_TAB_PROGRESS_ACTIVE ||
      !vte_terminal_get_termprop_uint_by_id (VTE_TERMINAL (self->terminal),
                                             VTE_PROPERTY_ID_PROGRESS_VALUE,
                                             &value))
    return .0;

  return MIN (value, 100) / 100.0;
}

G_GNUC_BEGIN_IGNORE_DEPRECATIONS
static void
draw_progress (cairo_t         *cr,
               GtkStyleContext *style_context,
               int              width,
               int              height,
               double           progress)
{
  GdkRGBA rgba;
  double alpha;

  g_assert (cr != NULL);
  g_assert (style_context != NULL);

  progress = CLAMP (progress, 0, 1);

  gtk_style_context_get_color (style_context, &rgba);

  alpha = rgba.alpha;
  rgba.alpha *= .15;
  gdk_cairo_set_source_rgba (cr, &rgba);

  cairo_arc (cr,
             width / 2,
             height / 2,
             width / 2,
             0.0,
             2 * M_PI);
  cairo_fill (cr);

  if (progress > 0.0)
    {
      rgba.alpha = alpha;
      gdk_cairo_set_source_rgba (cr, &rgba);

      cairo_arc (cr,
                 width / 2,
                 height / 2,
                 width / 2,
                 (-.5 * M_PI),
                 (2 * progress * M_PI) - (.5 * M_PI));

      if (progress != 1.0)
        {
          cairo_line_to (cr, width / 2, height / 2);
          cairo_line_to (cr, width / 2, 0);
        }

      cairo_fill (cr);
    }
}
G_GNUC_END_IGNORE_DEPRECATIONS

/**
 * bevy_tab_dup_indicator_icon:
 * @self: a #BevyTab
 *
 * Gets the progress indicator icon.
 *
 * Due to libadwaita not providing a way to do progress natively (as of 1.6)
 * this uses indicator icon to generate a progress icon using a drawing.
 *
 * Returns: (transfer full) (nullable): a #GIcon or %NULL
 */
GIcon *
bevy_tab_dup_indicator_icon (BevyTab *self)
{
  BevyTabProgress progress;

  g_return_val_if_fail (BEVY_IS_TAB (self), NULL);

  progress = bevy_tab_get_progress (self);

  if (progress == BEVY_TAB_PROGRESS_ERROR)
    return g_themed_icon_new ("dialog-error-symbolic");

  if (progress == BEVY_TAB_PROGRESS_INDETERMINATE)
    return NULL;

  if (progress == BEVY_TAB_PROGRESS_ACTIVE)
    {
      g_autoptr(GdkTexture) texture = NULL;
      g_autoptr(GBytes) bytes = NULL;
      cairo_surface_t *surface;
      cairo_t *cr;
      double fraction;
      int stride;
      int scale;
      int width;
      int height;

      fraction = bevy_tab_get_progress_fraction (self);
      scale = gtk_widget_get_scale_factor (GTK_WIDGET (self));
      width = 16 * scale;
      height = 16 * scale;

      surface = cairo_image_surface_create (CAIRO_FORMAT_ARGB32, width, height);
      stride = cairo_image_surface_get_stride (surface);
      cr = cairo_create (surface);

      G_GNUC_BEGIN_IGNORE_DEPRECATIONS {
        GtkStyleContext *style_context = gtk_widget_get_style_context (GTK_WIDGET (self));
        draw_progress (cr, style_context, width, height, fraction);
      } G_GNUC_END_IGNORE_DEPRECATIONS

      cairo_destroy (cr);

      bytes = g_bytes_new (cairo_image_surface_get_data (surface), height * stride);
      texture = gdk_memory_texture_new (width, height, GDK_MEMORY_DEFAULT, bytes, stride);

      cairo_surface_destroy (surface);

      return G_ICON (g_steal_pointer (&texture));
    }

  return NULL;
}

gboolean
bevy_tab_get_ignore_osc_title (BevyTab *self)
{
  g_return_val_if_fail (BEVY_IS_TAB (self), FALSE);

  return self->ignore_osc_title;
}

void
bevy_tab_set_ignore_osc_title (BevyTab *self,
                                 gboolean   ignore_osc_title)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  ignore_osc_title = !!ignore_osc_title;

  if (ignore_osc_title != self->ignore_osc_title)
    {
      self->ignore_osc_title = ignore_osc_title;
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_IGNORE_OSC_TITLE]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TITLE]);
    }
}

void
_bevy_tab_ignore_snapshot (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  self->ignore_snapshot = TRUE;
}

void
bevy_tab_grab_focus (BevyTab *self)
{
  g_return_if_fail (BEVY_IS_TAB (self));

  gtk_widget_grab_focus (GTK_WIDGET (self->terminal));
}
