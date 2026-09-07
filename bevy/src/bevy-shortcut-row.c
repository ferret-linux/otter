/*
 * bevy-shortcut-row.c
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

#include "bevy-shortcut-accel-dialog.h"
#include "bevy-shortcut-row.h"
#include "bevy-shortcuts.h"

struct _BevyShortcutRow
{
  AdwActionRow      parent_instance;

  char             *accelerator;
  char             *shortcut_name;

  AdwShortcutLabel *label;
  GtkStack         *stack;
};

enum {
  PROP_0,
  PROP_ACCELERATOR,
  PROP_SHORTCUT_NAME,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyShortcutRow, bevy_shortcut_row, ADW_TYPE_ACTION_ROW)

static GParamSpec *properties [N_PROPS];

static void
bevy_shortcut_row_dialog_shortcut_set_cb (BevyShortcutRow         *self,
                                            const char                *accelerator,
                                            BevyShortcutAccelDialog *dialog)
{
  g_assert (BEVY_IS_SHORTCUT_ROW (self));
  g_assert (BEVY_IS_SHORTCUT_ACCEL_DIALOG (dialog));

  bevy_shortcut_row_set_accelerator (self, accelerator ? accelerator : "");
}

static void
bevy_shortcut_row_select_shortcut (GtkWidget  *widget,
                                     const char *action,
                                     GVariant   *param)
{
  BevyShortcutRow *self = (BevyShortcutRow *)widget;
  g_autofree char *default_accelerator = NULL;
  BevyShortcutAccelDialog *dialog;
  const char *title;

  g_assert (BEVY_IS_SHORTCUT_ROW (self));

  title = adw_preferences_row_get_title (ADW_PREFERENCES_ROW (self));

  if (self->shortcut_name != NULL)
    default_accelerator = bevy_shortcuts_get_default_accelerator (self->shortcut_name);

  dialog = g_object_new (BEVY_TYPE_SHORTCUT_ACCEL_DIALOG,
                         "accelerator", self->accelerator,
                         "shortcut-title", title,
                         "default-accelerator", default_accelerator,
                         "title", _("Set Shortcut"),
                         NULL);
  g_signal_connect_object (dialog,
                           "shortcut-set",
                           G_CALLBACK (bevy_shortcut_row_dialog_shortcut_set_cb),
                           self,
                           G_CONNECT_SWAPPED);
  adw_dialog_present (ADW_DIALOG (dialog), GTK_WIDGET (gtk_widget_get_root (widget)));
}

static void
bevy_shortcut_row_dispose (GObject *object)
{
  BevyShortcutRow *self = (BevyShortcutRow *)object;

  gtk_widget_dispose_template (GTK_WIDGET (self), BEVY_TYPE_SHORTCUT_ROW);

  g_clear_pointer (&self->accelerator, g_free);
  g_clear_pointer (&self->shortcut_name, g_free);

  G_OBJECT_CLASS (bevy_shortcut_row_parent_class)->dispose (object);
}

static void
bevy_shortcut_row_get_property (GObject    *object,
                                  guint       prop_id,
                                  GValue     *value,
                                  GParamSpec *pspec)
{
  BevyShortcutRow *self = BEVY_SHORTCUT_ROW (object);

  switch (prop_id)
    {
    case PROP_ACCELERATOR:
      g_value_set_string (value, bevy_shortcut_row_get_accelerator (self));
      break;

    case PROP_SHORTCUT_NAME:
      g_value_set_string (value, bevy_shortcut_row_get_shortcut_name (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_shortcut_row_set_property (GObject      *object,
                                  guint         prop_id,
                                  const GValue *value,
                                  GParamSpec   *pspec)
{
  BevyShortcutRow *self = BEVY_SHORTCUT_ROW (object);

  switch (prop_id)
    {
    case PROP_ACCELERATOR:
      bevy_shortcut_row_set_accelerator (self, g_value_get_string (value));
      break;

    case PROP_SHORTCUT_NAME:
      bevy_shortcut_row_set_shortcut_name (self, g_value_get_string (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_shortcut_row_class_init (BevyShortcutRowClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->dispose = bevy_shortcut_row_dispose;
  object_class->get_property = bevy_shortcut_row_get_property;
  object_class->set_property = bevy_shortcut_row_set_property;

  properties[PROP_ACCELERATOR] =
    g_param_spec_string ("accelerator", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_SHORTCUT_NAME] =
    g_param_spec_string ("shortcut-name", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  gtk_widget_class_set_template_from_resource (widget_class, "/dev/itznoel/Bevy/bevy-shortcut-row.ui");
  gtk_widget_class_bind_template_child (widget_class, BevyShortcutRow, label);
  gtk_widget_class_bind_template_child (widget_class, BevyShortcutRow, stack);
  gtk_widget_class_install_action (widget_class, "shortcut.select", NULL, bevy_shortcut_row_select_shortcut);
}

static void
bevy_shortcut_row_init (BevyShortcutRow *self)
{
  gtk_widget_init_template (GTK_WIDGET (self));
}

const char *
bevy_shortcut_row_get_accelerator (BevyShortcutRow *self)
{
  g_return_val_if_fail (BEVY_IS_SHORTCUT_ROW (self), NULL);

  return self->accelerator;
}

void
bevy_shortcut_row_set_accelerator (BevyShortcutRow *self,
                                     const char        *accelerator)
{
  g_return_if_fail (BEVY_IS_SHORTCUT_ROW (self));

  if (g_set_str (&self->accelerator, accelerator))
    {
      gboolean disabled = TRUE;
      GdkModifierType state;
      guint keyval;

      if (accelerator && accelerator[0] && gtk_accelerator_parse (accelerator, &keyval, &state))
        disabled = FALSE;

      adw_shortcut_label_set_accelerator (self->label, accelerator);

      if (disabled)
        gtk_stack_set_visible_child_name (self->stack, "disabled");
      else
        gtk_stack_set_visible_child_name (self->stack, "label");

      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ACCELERATOR]);
    }
}

const char *
bevy_shortcut_row_get_shortcut_name (BevyShortcutRow *self)
{
  g_return_val_if_fail (BEVY_IS_SHORTCUT_ROW (self), NULL);

  return self->shortcut_name;
}

void
bevy_shortcut_row_set_shortcut_name (BevyShortcutRow *self,
                                       const char        *shortcut_name)
{
  g_return_if_fail (BEVY_IS_SHORTCUT_ROW (self));

  if (g_set_str (&self->shortcut_name, shortcut_name))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_SHORTCUT_NAME]);
}
