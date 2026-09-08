/*
 * bevy-shrinker.c
 *
 * Copyright 2024 Christian Hergert <chergert@redhat.com>
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


#include "bevy-shrinker.h"

struct _BevyShrinker
{
  GtkWidget parent_instance;
};

G_DEFINE_FINAL_TYPE (BevyShrinker, bevy_shrinker, GTK_TYPE_WIDGET)

static void
bevy_shrinker_measure (GtkWidget      *widget,
                         GtkOrientation  orientation,
                         int             for_size,
                         int            *min,
                         int            *nat,
                         int            *min_base,
                         int            *nat_base)
{
  GtkWidget *child;

  if ((child = gtk_widget_get_first_child (widget)))
    gtk_widget_measure (child, orientation, for_size, min, nat, min_base, nat_base);

  *nat = *min;
  *nat_base = *min_base;
}

static void
bevy_shrinker_size_allocate (GtkWidget *widget,
                               int        width,
                               int        height,
                               int        baseline)
{
  GtkWidget *child;

  if ((child = gtk_widget_get_first_child (widget)))
    gtk_widget_size_allocate (child, &(GtkAllocation) {0, 0, width, height}, baseline);
}

static void
bevy_shrinker_dispose (GObject *object)
{
  GtkWidget *child;

  while ((child = gtk_widget_get_first_child (GTK_WIDGET (object))))
    gtk_widget_unparent (child);

  G_OBJECT_CLASS (bevy_shrinker_parent_class)->dispose (object);
}

static void
bevy_shrinker_class_init (BevyShrinkerClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->dispose = bevy_shrinker_dispose;

  widget_class->measure = bevy_shrinker_measure;
  widget_class->size_allocate = bevy_shrinker_size_allocate;
}

static void
bevy_shrinker_init (BevyShrinker *self)
{
}
