/*
 * bevy-menu-row.c
 *
 * Copyright 2025 Christian Hergert <chergert@redhat.com>
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

#include "bevy-agent-ipc.h"
#include "bevy-menu-row.h"
#include "bevy-profile.h"

struct _BevyMenuRow
{
  GtkWidget  parent_instance;

  gpointer   item;
  GBinding  *binding;

  GtkLabel  *label;
};

enum {
  PROP_0,
  PROP_ITEM,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyMenuRow, bevy_menu_row, GTK_TYPE_WIDGET)

static GParamSpec *properties[N_PROPS];

static void
bevy_menu_row_dispose (GObject *object)
{
  BevyMenuRow *self = (BevyMenuRow *)object;
  GtkWidget *child;

  gtk_widget_dispose_template (GTK_WIDGET (self), BEVY_TYPE_MENU_ROW);

  while ((child = gtk_widget_get_first_child (GTK_WIDGET (self))))
    gtk_widget_unparent (child);

  g_clear_object (&self->binding);
  g_clear_object (&self->item);

  G_OBJECT_CLASS (bevy_menu_row_parent_class)->dispose (object);
}

static void
bevy_menu_row_get_property (GObject    *object,
                              guint       prop_id,
                              GValue     *value,
                              GParamSpec *pspec)
{
  BevyMenuRow *self = BEVY_MENU_ROW (object);

  switch (prop_id)
    {
    case PROP_ITEM:
      g_value_set_object (value, bevy_menu_row_get_item (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_menu_row_set_property (GObject      *object,
                              guint         prop_id,
                              const GValue *value,
                              GParamSpec   *pspec)
{
  BevyMenuRow *self = BEVY_MENU_ROW (object);

  switch (prop_id)
    {
    case PROP_ITEM:
      bevy_menu_row_set_item (self, g_value_get_object (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_menu_row_class_init (BevyMenuRowClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->dispose = bevy_menu_row_dispose;
  object_class->get_property = bevy_menu_row_get_property;
  object_class->set_property = bevy_menu_row_set_property;

  properties[PROP_ITEM] =
    g_param_spec_object ("item", NULL, NULL,
                         G_TYPE_OBJECT,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  gtk_widget_class_set_template_from_resource (widget_class, "/dev/itznoel/Bevy/bevy-menu-row.ui");
  gtk_widget_class_set_layout_manager_type (widget_class, GTK_TYPE_BIN_LAYOUT);
  gtk_widget_class_bind_template_child (widget_class, BevyMenuRow, label);
}

static void
bevy_menu_row_init (BevyMenuRow *self)
{
  gtk_widget_init_template (GTK_WIDGET (self));
}

gpointer
bevy_menu_row_get_item (BevyMenuRow *self)
{
  g_return_val_if_fail (BEVY_IS_MENU_ROW (self), NULL);

  return self->item;
}

void
bevy_menu_row_set_item (BevyMenuRow *self,
                          gpointer       item)
{
  g_return_if_fail (BEVY_IS_MENU_ROW (self));
  g_return_if_fail (!item || G_IS_OBJECT (item));

  if (self->item == item)
    return;

  if (self->binding)
    {
      g_binding_unbind (self->binding);
      g_clear_object (&self->binding);
    }

  g_set_object (&self->item, item);

  if (BEVY_IPC_IS_CONTAINER (self->item))
    {
      const char *id = bevy_ipc_container_get_id (self->item);

      if (g_strcmp0 (id, "session") == 0)
        gtk_label_set_label (self->label, _("My Computer"));
      else
        gtk_label_set_label (self->label,
                             bevy_ipc_container_get_display_name (self->item));

    }
  else if (BEVY_IS_PROFILE (self->item))
    {
      self->binding = g_object_ref (g_object_bind_property (self->item, "label", self->label, "label", G_BINDING_SYNC_CREATE));
    }

  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ITEM]);
}
