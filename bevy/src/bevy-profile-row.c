/*
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

#include "bevy-application.h"
#include "bevy-preferences-window.h"
#include "bevy-profile-editor.h"
#include "bevy-profile-row.h"

struct _BevyProfileRow
{
  AdwActionRow    parent_instance;

  GtkImage       *checkmark;

  BevyProfile *profile;
};

enum {
  PROP_0,
  PROP_PROFILE,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyProfileRow, bevy_profile_row, ADW_TYPE_ACTION_ROW)

static GParamSpec *properties [N_PROPS];

static void
bevy_profile_row_duplicate (GtkWidget  *widget,
                              const char *action_name,
                              GVariant   *param)
{
  BevyProfileRow *self = (BevyProfileRow *)widget;
  g_autoptr(BevyProfile) profile = NULL;

  g_assert (BEVY_IS_PROFILE_ROW (self));

  profile = bevy_profile_duplicate (self->profile);
}

G_GNUC_BEGIN_IGNORE_DEPRECATIONS

static void
bevy_profile_row_edit (GtkWidget  *widget,
                         const char *action_name,
                         GVariant   *param)
{
  AdwPreferencesWindow *window;
  BevyProfileEditor *editor;
  BevyProfileRow *self = (BevyProfileRow *)widget;

  g_assert (BEVY_IS_PROFILE_ROW (self));

  window = ADW_PREFERENCES_WINDOW (gtk_widget_get_ancestor (widget, ADW_TYPE_PREFERENCES_WINDOW));
  editor = bevy_profile_editor_new (self->profile);

  adw_preferences_window_pop_subpage (ADW_PREFERENCES_WINDOW (window));
  adw_preferences_window_push_subpage (ADW_PREFERENCES_WINDOW (window),
                                       ADW_NAVIGATION_PAGE (editor));
}

static void
bevy_profile_row_undo_clicked_cb (AdwToast      *toast,
                                    BevyProfile *profile)
{
  g_assert (ADW_IS_TOAST (toast));
  g_assert (BEVY_IS_PROFILE (profile));

  bevy_application_add_profile (BEVY_APPLICATION_DEFAULT, profile);
}

static void
bevy_profile_row_remove (GtkWidget  *widget,
                           const char *action_name,
                           GVariant   *param)
{
  BevyProfileRow *self = (BevyProfileRow *)widget;
  AdwPreferencesWindow *window;
  AdwToast *toast;

  g_assert (BEVY_IS_PROFILE_ROW (self));

  window = ADW_PREFERENCES_WINDOW (gtk_widget_get_ancestor (widget, ADW_TYPE_PREFERENCES_WINDOW));
  toast = adw_toast_new_format (_("Removed profile “%s”"),
                                bevy_profile_dup_label (self->profile));
  adw_toast_set_button_label (toast, _("Undo"));
  g_signal_connect_data (toast,
                         "button-clicked",
                         G_CALLBACK (bevy_profile_row_undo_clicked_cb),
                         g_object_ref (self->profile),
                         (GClosureNotify)g_object_unref,
                         0);

  bevy_application_remove_profile (BEVY_APPLICATION_DEFAULT, self->profile);

  adw_preferences_window_add_toast (window, toast);
}

G_GNUC_END_IGNORE_DEPRECATIONS

static void
bevy_profile_row_make_default (GtkWidget  *widget,
                                 const char *action_name,
                                 GVariant   *param)
{
  BevyProfileRow *self = BEVY_PROFILE_ROW (widget);

  bevy_application_set_default_profile (BEVY_APPLICATION_DEFAULT, self->profile);
}

static void
bevy_profile_row_default_profile_changed_cb (BevyProfileRow *self,
                                               GParamSpec       *pspec,
                                               BevySettings   *settings)
{
  const char *default_uuid;
  gboolean is_default;

  g_assert (BEVY_IS_PROFILE_ROW (self));
  g_assert (BEVY_IS_SETTINGS (settings));

  default_uuid = bevy_settings_dup_default_profile_uuid (settings);

  is_default = g_strcmp0 (default_uuid, bevy_profile_get_uuid (self->profile)) == 0;

  gtk_widget_set_visible (GTK_WIDGET (self->checkmark), is_default);
}

static void
bevy_profile_row_constructed (GObject *object)
{
  BevyProfileRow *self = (BevyProfileRow *)object;
  BevyApplication *app = BEVY_APPLICATION_DEFAULT;
  BevySettings *settings = bevy_application_get_settings (app);

  G_OBJECT_CLASS (bevy_profile_row_parent_class)->constructed (object);

  g_signal_connect_object (settings,
                           "notify::default-profile-uuid",
                           G_CALLBACK (bevy_profile_row_default_profile_changed_cb),
                           self,
                           G_CONNECT_SWAPPED);

  bevy_profile_row_default_profile_changed_cb (self, NULL, settings);

  g_object_bind_property (self->profile, "label", self, "title",
                          G_BINDING_SYNC_CREATE);
}

static void
bevy_profile_row_dispose (GObject *object)
{
  BevyProfileRow *self = (BevyProfileRow *)object;

  gtk_widget_dispose_template (GTK_WIDGET (self), BEVY_TYPE_PROFILE_ROW);

  g_clear_object (&self->profile);

  G_OBJECT_CLASS (bevy_profile_row_parent_class)->dispose (object);
}

static void
bevy_profile_row_get_property (GObject    *object,
                                 guint       prop_id,
                                 GValue     *value,
                                 GParamSpec *pspec)
{
  BevyProfileRow *self = BEVY_PROFILE_ROW (object);

  switch (prop_id)
    {
    case PROP_PROFILE:
      g_value_set_object (value, bevy_profile_row_get_profile (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_profile_row_set_property (GObject      *object,
                                 guint         prop_id,
                                 const GValue *value,
                                 GParamSpec   *pspec)
{
  BevyProfileRow *self = BEVY_PROFILE_ROW (object);

  switch (prop_id)
    {
    case PROP_PROFILE:
      self->profile = g_value_dup_object (value);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_profile_row_class_init (BevyProfileRowClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->constructed = bevy_profile_row_constructed;
  object_class->dispose = bevy_profile_row_dispose;
  object_class->get_property = bevy_profile_row_get_property;
  object_class->set_property = bevy_profile_row_set_property;

  properties[PROP_PROFILE] =
    g_param_spec_object ("profile", NULL, NULL,
                         BEVY_TYPE_PROFILE,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  gtk_widget_class_install_action (widget_class,
                                   "profile.duplicate",
                                   NULL,
                                   bevy_profile_row_duplicate);
  gtk_widget_class_install_action (widget_class,
                                   "profile.edit",
                                   NULL,
                                   bevy_profile_row_edit);
  gtk_widget_class_install_action (widget_class,
                                   "profile.remove",
                                   NULL,
                                   bevy_profile_row_remove);
  gtk_widget_class_install_action (widget_class,
                                   "profile.make-default",
                                   NULL,
                                   bevy_profile_row_make_default);

  gtk_widget_class_set_template_from_resource (widget_class, "/dev/itznoel/Bevy/bevy-profile-row.ui");

  gtk_widget_class_bind_template_child (widget_class, BevyProfileRow, checkmark);
}

static void
bevy_profile_row_init (BevyProfileRow *self)
{
  gtk_widget_init_template (GTK_WIDGET (self));
}

GtkWidget *
bevy_profile_row_new (BevyProfile *profile)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (profile), NULL);

  return g_object_new (BEVY_TYPE_PROFILE_ROW,
                       "profile", profile,
                       NULL);
}

BevyProfile *
bevy_profile_row_get_profile (BevyProfileRow *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE_ROW (self), NULL);

  return self->profile;
}
