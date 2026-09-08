/* bevy-container-provider.c
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


#include "bevy-container-provider.h"

typedef struct
{
  GPtrArray *containers;
} BevyContainerProviderPrivate;

static void list_model_iface_init (GListModelInterface *iface);

G_DEFINE_ABSTRACT_TYPE_WITH_CODE (BevyContainerProvider, bevy_container_provider, G_TYPE_OBJECT,
                                  G_ADD_PRIVATE (BevyContainerProvider)
                                  G_IMPLEMENT_INTERFACE (G_TYPE_LIST_MODEL, list_model_iface_init))

enum {
  ADDED,
  REMOVED,
  N_SIGNALS
};

static guint signals[N_SIGNALS];

static void
bevy_container_provider_real_added (BevyContainerProvider *self,
                                      BevyIpcContainer      *container)
{
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);
  guint position;

  g_assert (BEVY_IS_CONTAINER_PROVIDER (self));
  g_assert (BEVY_IPC_IS_CONTAINER (container));

  g_debug ("Added container \"%s\"",
           bevy_ipc_container_get_id (container));

  position = priv->containers->len;

  g_ptr_array_add (priv->containers, g_object_ref (container));
  g_list_model_items_changed (G_LIST_MODEL (self), position, 0, 1);
}

static void
bevy_container_provider_real_removed (BevyContainerProvider *self,
                                        BevyIpcContainer      *container)
{
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);
  guint position;

  g_assert (BEVY_IS_CONTAINER_PROVIDER (self));
  g_assert (BEVY_IPC_IS_CONTAINER (container));

  g_debug ("Removed container \"%s\"",
           bevy_ipc_container_get_id (container));

  if (g_ptr_array_find (priv->containers, container, &position))
    {
      g_ptr_array_remove_index (priv->containers, position);
      g_list_model_items_changed (G_LIST_MODEL (self), position, 1, 0);
    }
}

static void
bevy_container_provider_dispose (GObject *object)
{
  BevyContainerProvider *self = (BevyContainerProvider *)object;
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  if (priv->containers->len > 0)
    g_ptr_array_remove_range (priv->containers, 0, priv->containers->len);

  G_OBJECT_CLASS (bevy_container_provider_parent_class)->dispose (object);
}

static void
bevy_container_provider_finalize (GObject *object)
{
  BevyContainerProvider *self = (BevyContainerProvider *)object;
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  g_clear_pointer (&priv->containers, g_ptr_array_unref);

  G_OBJECT_CLASS (bevy_container_provider_parent_class)->finalize (object);
}

static void
bevy_container_provider_class_init (BevyContainerProviderClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->dispose = bevy_container_provider_dispose;
  object_class->finalize = bevy_container_provider_finalize;

  klass->added = bevy_container_provider_real_added;
  klass->removed = bevy_container_provider_real_removed;

  signals[ADDED] =
    g_signal_new ("added",
                  G_TYPE_FROM_CLASS (klass),
                  G_SIGNAL_RUN_LAST,
                  G_STRUCT_OFFSET (BevyContainerProviderClass, added),
                  NULL, NULL,
                  NULL,
                  G_TYPE_NONE, 1, BEVY_IPC_TYPE_CONTAINER);

  signals[REMOVED] =
    g_signal_new ("removed",
                  G_TYPE_FROM_CLASS (klass),
                  G_SIGNAL_RUN_LAST,
                  G_STRUCT_OFFSET (BevyContainerProviderClass, removed),
                  NULL, NULL,
                  NULL,
                  G_TYPE_NONE, 1, BEVY_IPC_TYPE_CONTAINER);
}

static void
bevy_container_provider_init (BevyContainerProvider *self)
{
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  priv->containers = g_ptr_array_new_with_free_func (g_object_unref);
}

void
bevy_container_provider_emit_added (BevyContainerProvider *self,
                                      BevyIpcContainer      *container)
{
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);
  const char *id;

  g_return_if_fail (BEVY_IS_CONTAINER_PROVIDER (self));
  g_return_if_fail (BEVY_IPC_IS_CONTAINER (container));

  id = bevy_ipc_container_get_id (container);

  for (guint i = 0; i < priv->containers->len; i++)
    {
      if (g_strcmp0 (id, bevy_ipc_container_get_id (g_ptr_array_index (priv->containers, i))) == 0)
        {
          g_warning ("Container \"%s\" already added", id);
          return;
        }
    }

  g_signal_emit (self, signals[ADDED], 0, container);
}

void
bevy_container_provider_emit_removed (BevyContainerProvider *self,
                                        BevyIpcContainer      *container)
{
  g_return_if_fail (BEVY_IS_CONTAINER_PROVIDER (self));
  g_return_if_fail (BEVY_IPC_IS_CONTAINER (container));

  g_signal_emit (self, signals[REMOVED], 0, container);
}

static gboolean
compare_by_id (gconstpointer a,
               gconstpointer b)
{
  return 0 == g_strcmp0 (bevy_ipc_container_get_id ((BevyIpcContainer *)a),
                         bevy_ipc_container_get_id ((BevyIpcContainer *)b));
}

void
bevy_container_provider_merge (BevyContainerProvider *self,
                                 GPtrArray               *containers)
{
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  g_return_if_fail (BEVY_IS_CONTAINER_PROVIDER (self));
  g_return_if_fail (containers != NULL);

  /* First remove any containers not in the set, or replace them with
   * the new version of the object. Scan in reverse so that we can
   * have stable indexes.
   */
  for (guint i = priv->containers->len; i > 0; i--)
    {
      BevyIpcContainer *container = g_ptr_array_index (priv->containers, i-1);
      guint position;

      if (g_ptr_array_find_with_equal_func (containers, container, compare_by_id, &position))
        {
          g_ptr_array_index (priv->containers, i-1) = g_object_ref (g_ptr_array_index (containers, position));
          g_list_model_items_changed (G_LIST_MODEL (self), i-1, 1, 1);
          continue;
        }

      bevy_container_provider_emit_removed (self, container);
    }

  for (guint i = 0; i < containers->len; i++)
    {
      BevyIpcContainer *container = g_ptr_array_index (containers, i);
      guint position;

      if (!g_ptr_array_find_with_equal_func (priv->containers, container, compare_by_id, &position))
        bevy_container_provider_emit_added (self, container);
    }
}

static GType
bevy_container_provider_get_item_type (GListModel *model)
{
  return BEVY_IPC_TYPE_CONTAINER;
}

static guint
bevy_container_provider_get_n_items (GListModel *model)
{
  BevyContainerProvider *self = BEVY_CONTAINER_PROVIDER (model);
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  return priv->containers->len;
}

static gpointer
bevy_container_provider_get_item (GListModel *model,
                                    guint       position)
{
  BevyContainerProvider *self = BEVY_CONTAINER_PROVIDER (model);
  BevyContainerProviderPrivate *priv = bevy_container_provider_get_instance_private (self);

  if (position < priv->containers->len)
    return g_object_ref (g_ptr_array_index (priv->containers, position));

  return NULL;
}

static void
list_model_iface_init (GListModelInterface *iface)
{
  iface->get_item_type = bevy_container_provider_get_item_type;
  iface->get_n_items = bevy_container_provider_get_n_items;
  iface->get_item = bevy_container_provider_get_item;
}
