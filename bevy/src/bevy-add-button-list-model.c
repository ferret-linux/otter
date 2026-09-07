/* bevy-add-button-list-model.c
 *
 * Copyright 2025 Marco Mastropaolo <marco@mastropaolo.com>
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

#include "bevy-add-button-list-model.h"
#include "bevy-add-button-list-item.h"

struct _BevyAddButtonListModel
{
  GObject     parent_instance;

  GListModel *model;
  GType       item_type;
};

enum {
  PROP_0,
  PROP_MODEL,
  N_PROPS
};

static GParamSpec *properties[N_PROPS];

static void list_model_iface_init (GListModelInterface *iface);

G_DEFINE_FINAL_TYPE_WITH_CODE (BevyAddButtonListModel, bevy_add_button_list_model, G_TYPE_OBJECT,
                               G_IMPLEMENT_INTERFACE (G_TYPE_LIST_MODEL, list_model_iface_init))

static void
bevy_add_button_list_model_items_changed_cb (BevyAddButtonListModel *self,
                                               guint                     position,
                                               guint                     removed,
                                               guint                     added,
                                               GListModel               *model)
{
  g_list_model_items_changed (G_LIST_MODEL (self), position, removed, added);
}

static void
bevy_add_button_list_model_constructed (GObject *object)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (object);

  G_OBJECT_CLASS (bevy_add_button_list_model_parent_class)->constructed (object);

  if (self->model)
    {
      g_signal_connect_object (self->model,
                               "items-changed",
                               G_CALLBACK (bevy_add_button_list_model_items_changed_cb),
                               self,
                               G_CONNECT_SWAPPED);

      self->item_type = g_list_model_get_item_type (self->model);
    }
  else
    {
      self->item_type = G_TYPE_OBJECT;
    }
}

static void
bevy_add_button_list_model_dispose (GObject *object)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (object);

  g_clear_object (&self->model);

  G_OBJECT_CLASS (bevy_add_button_list_model_parent_class)->dispose (object);
}

static void
bevy_add_button_list_model_get_property (GObject    *object,
                                           guint       prop_id,
                                           GValue     *value,
                                           GParamSpec *pspec)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (object);

  switch (prop_id)
    {
    case PROP_MODEL:
      g_value_set_object (value, self->model);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_add_button_list_model_set_property (GObject      *object,
                                           guint         prop_id,
                                           const GValue *value,
                                           GParamSpec   *pspec)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (object);

  switch (prop_id)
    {
    case PROP_MODEL:
      g_set_object (&self->model, g_value_get_object (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_add_button_list_model_class_init (BevyAddButtonListModelClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->constructed = bevy_add_button_list_model_constructed;
  object_class->dispose = bevy_add_button_list_model_dispose;
  object_class->get_property = bevy_add_button_list_model_get_property;
  object_class->set_property = bevy_add_button_list_model_set_property;

  properties[PROP_MODEL] =
    g_param_spec_object ("model", NULL, NULL,
                         G_TYPE_LIST_MODEL,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);
}

static void
bevy_add_button_list_model_init (BevyAddButtonListModel *self)
{
  self->item_type = G_TYPE_OBJECT;
}

static GType
bevy_add_button_list_model_get_item_type (GListModel *model)
{
  return BEVY_TYPE_ADD_BUTTON_LIST_ITEM;
}

static guint
bevy_add_button_list_model_get_n_items (GListModel *model)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (model);

  if (self->model)
    return g_list_model_get_n_items (self->model) + 1;
  else
    return 1;
}

static gpointer
bevy_add_button_list_model_get_item (GListModel *model,
                                       guint       position)
{
  BevyAddButtonListModel *self = BEVY_ADD_BUTTON_LIST_MODEL (model);
  guint n_items = 0;

  if (self->model)
    n_items = g_list_model_get_n_items (self->model);

  if (position < n_items)
    {
      g_autoptr(GObject) item = g_list_model_get_item (self->model, position);
      return bevy_add_button_list_item_new (item);
    }
  else if (position == n_items)
    {
      return bevy_add_button_list_item_new (NULL);
    }

  return NULL;
}

static void
list_model_iface_init (GListModelInterface *iface)
{
  iface->get_item_type = bevy_add_button_list_model_get_item_type;
  iface->get_n_items = bevy_add_button_list_model_get_n_items;
  iface->get_item = bevy_add_button_list_model_get_item;
}

BevyAddButtonListModel *
bevy_add_button_list_model_new (GListModel *model)
{
  return g_object_new (BEVY_TYPE_ADD_BUTTON_LIST_MODEL,
                       "model", model,
                       NULL);
}

GListModel *
bevy_add_button_list_model_get_model (BevyAddButtonListModel *self)
{
  g_return_val_if_fail (BEVY_IS_ADD_BUTTON_LIST_MODEL (self), NULL);

  return self->model;
}
