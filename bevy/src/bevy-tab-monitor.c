/*
 * bevy-tab-monitor.c
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


#include "bevy-tab-monitor.h"

#define DELAY_INTERACTIVE_MSEC 100
#define DELAY_MIN_MSEC         500
#define DELAY_MAX_MSEC         10000

struct _BevyTabMonitor
{
  GObject   parent_instance;
  GWeakRef  tab_wr;
  GSource  *update_source;
  int       current_delay_msec;
  guint     has_pressed_key : 1;
  guint     is_polling : 1;
};

enum {
  PROP_0,
  PROP_TAB,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyTabMonitor, bevy_tab_monitor, G_TYPE_OBJECT)

static GParamSpec *properties[N_PROPS];

static gint64
bevy_tab_monitor_get_ready_time (BevyTabMonitor *self)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));

  /* Less than a second, we just be precise to keep this easy. */
  if (self->current_delay_msec < 1000)
    return g_get_monotonic_time () + (self->current_delay_msec * 1000);

  /* A second or more, we want to try to align things with
   * other tabs so we just wake up once and poll them all.
   */
  return (g_get_monotonic_time () / G_USEC_PER_SEC * G_USEC_PER_SEC) + (self->current_delay_msec * 1000);
}

static void
bevy_tab_monitor_reset_delay (BevyTabMonitor *self)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));
  g_assert (self->update_source != NULL);

  self->current_delay_msec = DELAY_MIN_MSEC;
  g_source_set_ready_time (self->update_source,
                           bevy_tab_monitor_get_ready_time (self));
}

static void
bevy_tab_monitor_same_delay (BevyTabMonitor *self)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));

  g_source_set_ready_time (self->update_source,
                           bevy_tab_monitor_get_ready_time (self));
}

static void
bevy_tab_monitor_backoff_delay (BevyTabMonitor *self)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));
  g_assert (self->update_source != NULL);

  self->current_delay_msec = CLAMP (self->current_delay_msec * 2, DELAY_MIN_MSEC, DELAY_MAX_MSEC);
  g_source_set_ready_time (self->update_source,
                           bevy_tab_monitor_get_ready_time (self));
}

static void
bevy_tab_monitor_poll_agent_cb (GObject      *object,
                                  GAsyncResult *result,
                                  gpointer      user_data)
{
  BevyTab *tab = (BevyTab *)object;
  g_autoptr(BevyTabMonitor) self = user_data;

  g_assert (BEVY_IS_TAB (tab));
  g_assert (G_IS_ASYNC_RESULT (result));
  g_assert (BEVY_IS_TAB_MONITOR (self));

  self->is_polling = FALSE;

  if (self->update_source == NULL)
    return;

  if (bevy_tab_poll_agent_finish (tab, result, NULL))
    bevy_tab_monitor_reset_delay (self);
  else
    bevy_tab_monitor_backoff_delay (self);
}

static gboolean
bevy_tab_monitor_update_source_func (gpointer user_data)
{
  BevyTabMonitor *self = user_data;
  g_autoptr(BevyTab) tab = NULL;
  BevyIpcProcess *process;

  g_assert (BEVY_IS_TAB_MONITOR (self));

  if ((tab = g_weak_ref_get (&self->tab_wr)) &&
      (process = bevy_tab_get_process (tab)))
    {
      bevy_tab_monitor_same_delay (self);

      if (!self->is_polling)
        {
          self->is_polling = TRUE;

          bevy_tab_poll_agent_async (tab,
                                       NULL,
                                       bevy_tab_monitor_poll_agent_cb,
                                       g_object_ref (self));
        }

      return G_SOURCE_CONTINUE;
    }

  g_clear_pointer (&self->update_source, g_source_unref);

  return G_SOURCE_REMOVE;
}

static gboolean
bevy_tab_monitor_dispatch (GSource     *source,
                             GSourceFunc  callback,
                             gpointer     data)
{
  return callback (data);
}

static const GSourceFuncs source_funcs = {
  .dispatch = bevy_tab_monitor_dispatch,
};

static void
bevy_tab_monitor_queue_update (BevyTabMonitor *self)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));

  if G_UNLIKELY (self->update_source == NULL)
    {
      self->update_source = g_source_new ((GSourceFuncs *)&source_funcs, sizeof (GSource));
      g_source_set_callback (self->update_source,
                             bevy_tab_monitor_update_source_func,
                             self, NULL);
      g_source_set_static_name (self->update_source, "[bevy-tab-monitor]");
      g_source_set_priority (self->update_source, G_PRIORITY_LOW);
      bevy_tab_monitor_reset_delay (self);
      g_source_attach (self->update_source, NULL);
      return;
    }

  if G_UNLIKELY (self->current_delay_msec > DELAY_MIN_MSEC)
    {
      bevy_tab_monitor_reset_delay (self);
      return;
    }
}

static void
bevy_tab_monitor_terminal_contents_changed_cb (BevyTabMonitor *self,
                                                 BevyTerminal   *terminal)
{
  g_assert (BEVY_IS_TAB_MONITOR (self));
  g_assert (BEVY_IS_TERMINAL (terminal));

  bevy_tab_monitor_queue_update (self);
}

static gboolean
bevy_tab_monitor_key_pressed_cb (BevyTabMonitor      *self,
                                   guint                  keyval,
                                   guint                  keycode,
                                   GdkModifierType        state,
                                   GtkEventControllerKey *key)
{
  gboolean low_delay = FALSE;

  g_assert (BEVY_IS_TAB_MONITOR (self));
  g_assert (GTK_IS_EVENT_CONTROLLER_KEY (key));

  bevy_tab_monitor_set_has_pressed_key (self, TRUE);

  if (self->update_source == NULL)
    return GDK_EVENT_PROPAGATE;

  state &= gtk_accelerator_get_default_mod_mask ();

  switch (keyval)
    {
    case GDK_KEY_Return:
    case GDK_KEY_ISO_Enter:
    case GDK_KEY_KP_Enter:
      low_delay = TRUE;
      break;

    case GDK_KEY_c:
    case GDK_KEY_d:
      low_delay = !!(state & GDK_CONTROL_MASK);
      break;

    default:
      break;
    }

  if (low_delay)
    {
      self->current_delay_msec = DELAY_INTERACTIVE_MSEC;
      g_source_set_ready_time (self->update_source,
                               bevy_tab_monitor_get_ready_time (self));
    }

  return GDK_EVENT_PROPAGATE;
}

static void
bevy_tab_monitor_set_tab (BevyTabMonitor *self,
                            BevyTab        *tab)
{
  BevyTerminal *terminal;
  GtkEventController *controller;

  g_assert (BEVY_IS_TAB_MONITOR (self));
  g_assert (BEVY_IS_TAB (tab));

  g_weak_ref_set (&self->tab_wr, tab);

  terminal = bevy_tab_get_terminal (tab);

  g_signal_connect_object (terminal,
                           "contents-changed",
                           G_CALLBACK (bevy_tab_monitor_terminal_contents_changed_cb),
                           self,
                           G_CONNECT_SWAPPED);

  /* We use an input controller to sniff for certain keys which will make us
   * want to poll at a lower frequency than the delay. For example, something
   * like ctrl+d, enter, etc as *input* indicates that we could be making a
   * transition sooner.
   */
  controller = gtk_event_controller_key_new ();
  g_signal_connect_object (controller,
                           "key-pressed",
                           G_CALLBACK (bevy_tab_monitor_key_pressed_cb),
                           self,
                           G_CONNECT_SWAPPED);
  gtk_event_controller_set_propagation_phase (controller, GTK_PHASE_CAPTURE);
  gtk_widget_add_controller (GTK_WIDGET (tab), controller);
}

static void
bevy_tab_monitor_finalize (GObject *object)
{
  BevyTabMonitor *self = (BevyTabMonitor *)object;

  if (self->update_source != NULL)
    {
      g_source_destroy (self->update_source);
      g_clear_pointer (&self->update_source, g_source_unref);
    }

  g_weak_ref_clear (&self->tab_wr);

  G_OBJECT_CLASS (bevy_tab_monitor_parent_class)->finalize (object);
}

static void
bevy_tab_monitor_get_property (GObject    *object,
                                 guint       prop_id,
                                 GValue     *value,
                                 GParamSpec *pspec)
{
  BevyTabMonitor *self = BEVY_TAB_MONITOR (object);

  switch (prop_id)
    {
    case PROP_TAB:
      g_value_take_object (value, g_weak_ref_get (&self->tab_wr));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_tab_monitor_set_property (GObject      *object,
                                 guint         prop_id,
                                 const GValue *value,
                                 GParamSpec   *pspec)
{
  BevyTabMonitor *self = BEVY_TAB_MONITOR (object);

  switch (prop_id)
    {
    case PROP_TAB:
      bevy_tab_monitor_set_tab (self, g_value_get_object (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_tab_monitor_class_init (BevyTabMonitorClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->finalize = bevy_tab_monitor_finalize;
  object_class->get_property = bevy_tab_monitor_get_property;
  object_class->set_property = bevy_tab_monitor_set_property;

  properties[PROP_TAB] =
    g_param_spec_object ("tab", NULL, NULL,
                         BEVY_TYPE_TAB,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);
}

static void
bevy_tab_monitor_init (BevyTabMonitor *self)
{
  self->current_delay_msec = DELAY_MIN_MSEC;

  g_weak_ref_init (&self->tab_wr, NULL);
}

BevyTabMonitor *
bevy_tab_monitor_new (BevyTab *tab)
{
  g_return_val_if_fail (BEVY_IS_TAB (tab), NULL);

  return g_object_new (BEVY_TYPE_TAB_MONITOR,
                       "tab", tab,
                       NULL);
}

gboolean
bevy_tab_monitor_get_has_pressed_key (BevyTabMonitor *self)
{
  g_return_val_if_fail (BEVY_IS_TAB_MONITOR (self), FALSE);

  return self->has_pressed_key;
}

void
bevy_tab_monitor_set_has_pressed_key (BevyTabMonitor *self,
                                        gboolean          has_pressed_key)
{
  g_return_if_fail (BEVY_IS_TAB_MONITOR (self));

  self->has_pressed_key = has_pressed_key;
}
