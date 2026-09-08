/* bevy-palette-preview.c
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


#include "bevy-palette-preview.h"
#include "bevy-palette-preview-color.h"

struct _BevyPalettePreview
{
  GtkWidget parent_instance;

  BevyPalette *palette;
  PangoFontDescription *font_desc;

  GtkImage *image;
  GtkLabel *label;

  guint dark : 1;
  guint selected : 1;
};

enum {
  PROP_0,
  PROP_DARK,
  PROP_FONT_DESC,
  PROP_PALETTE,
  PROP_SELECTED,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyPalettePreview, bevy_palette_preview, GTK_TYPE_WIDGET)

static GParamSpec *properties [N_PROPS];

static void
bevy_palette_preview_update_label (BevyPalettePreview *self)
{
  g_autoptr(PangoAttrList) attrs = NULL;

  g_assert (BEVY_IS_PALETTE_PREVIEW (self));

  attrs = pango_attr_list_new ();

  if (self->font_desc != NULL)
    pango_attr_list_insert (attrs, pango_attr_font_desc_new (self->font_desc));

  if (self->palette != NULL)
    {
      const BevyPaletteFace *face = bevy_palette_get_face (self->palette, self->dark);
      const GdkRGBA *color = &face->foreground;

      pango_attr_list_insert (attrs,
                              pango_attr_foreground_new (color->red * 65535,
                                                         color->green * 65535,
                                                         color->blue * 65535));
    }

  gtk_label_set_attributes (self->label, attrs);
}

static void
bevy_palette_preview_snapshot (GtkWidget   *widget,
                                 GtkSnapshot *snapshot)
{
  BevyPalettePreview *self = (BevyPalettePreview *)widget;
  const BevyPaletteFace *face;
  int width;
  int height;

  g_assert (BEVY_IS_PALETTE_PREVIEW (self));

  if (self->palette == NULL)
    return;

  face = bevy_palette_get_face (self->palette, self->dark);
  width = gtk_widget_get_width (widget);
  height = gtk_widget_get_height (widget);

  gtk_snapshot_append_color (snapshot,
                             &face->background,
                             &GRAPHENE_RECT_INIT (0, 0, width, height));

  GTK_WIDGET_CLASS (bevy_palette_preview_parent_class)->snapshot (widget, snapshot);
}

static gboolean
bevy_palette_preview_query_tooltip (GtkWidget  *widget,
                                      int         x,
                                      int         y,
                                      gboolean    keyboard_mode,
                                      GtkTooltip *tooltip)
{
  BevyPalettePreview *self = (BevyPalettePreview *)widget;

  g_assert (BEVY_IS_PALETTE_PREVIEW (self));

  if (self->palette == NULL)
    return FALSE;

  gtk_tooltip_set_text (tooltip, bevy_palette_get_name (self->palette));

  return TRUE;
}

static void
bevy_palette_preview_constructed (GObject *object)
{
  BevyPalettePreview *self = (BevyPalettePreview *)object;

  G_OBJECT_CLASS (bevy_palette_preview_parent_class)->constructed (object);

  bevy_palette_preview_update_label (self);
}

static void
bevy_palette_preview_dispose (GObject *object)
{
  BevyPalettePreview *self = (BevyPalettePreview *)object;
  GtkWidget *child;

  gtk_widget_dispose_template (GTK_WIDGET (self), BEVY_TYPE_PALETTE_PREVIEW);

  while ((child = gtk_widget_get_first_child (GTK_WIDGET(self))))
    gtk_widget_unparent (child);

  g_clear_object (&self->palette);
  g_clear_pointer (&self->font_desc, pango_font_description_free);

  G_OBJECT_CLASS (bevy_palette_preview_parent_class)->dispose (object);
}

static void
bevy_palette_preview_get_property (GObject    *object,
                                     guint       prop_id,
                                     GValue     *value,
                                     GParamSpec *pspec)
{
  BevyPalettePreview *self = BEVY_PALETTE_PREVIEW (object);

  switch (prop_id)
    {
    case PROP_FONT_DESC:
      g_value_set_boxed (value, bevy_palette_preview_get_font_desc (self));
      break;

    case PROP_PALETTE:
      g_value_set_object (value, self->palette);
      break;

    case PROP_DARK:
      g_value_set_boolean (value, bevy_palette_preview_get_dark (self));
      break;

    case PROP_SELECTED:
      g_value_set_boolean (value, self->selected);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_palette_preview_set_property (GObject      *object,
                                     guint         prop_id,
                                     const GValue *value,
                                     GParamSpec   *pspec)
{
  BevyPalettePreview *self = BEVY_PALETTE_PREVIEW (object);

  switch (prop_id)
    {
    case PROP_FONT_DESC:
      bevy_palette_preview_set_font_desc (self, g_value_get_boxed (value));
      break;

    case PROP_PALETTE:
      self->palette = g_value_dup_object (value);
      break;

    case PROP_DARK:
      bevy_palette_preview_set_dark (self, g_value_get_boolean (value));
      break;

    case PROP_SELECTED:
      if (self->selected != g_value_get_boolean (value))
        {
          self->selected = g_value_get_boolean (value);
          g_object_notify_by_pspec (G_OBJECT (self), pspec);

          if (self->selected)
            gtk_widget_add_css_class (gtk_widget_get_parent (GTK_WIDGET (self)), "selected");
          else
            gtk_widget_remove_css_class (gtk_widget_get_parent (GTK_WIDGET (self)), "selected");
        }
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_palette_preview_class_init (BevyPalettePreviewClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);
  GtkWidgetClass *widget_class = GTK_WIDGET_CLASS (klass);

  object_class->constructed = bevy_palette_preview_constructed;
  object_class->dispose = bevy_palette_preview_dispose;
  object_class->get_property = bevy_palette_preview_get_property;
  object_class->set_property = bevy_palette_preview_set_property;

  widget_class->query_tooltip = bevy_palette_preview_query_tooltip;
  widget_class->snapshot = bevy_palette_preview_snapshot;

  properties[PROP_FONT_DESC] =
    g_param_spec_boxed ("font-desc", NULL, NULL,
                        PANGO_TYPE_FONT_DESCRIPTION,
                        (G_PARAM_READWRITE |
                         G_PARAM_EXPLICIT_NOTIFY |
                         G_PARAM_STATIC_STRINGS));

  properties[PROP_PALETTE] =
    g_param_spec_object ("palette", NULL, NULL,
                         BEVY_TYPE_PALETTE,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_DARK] =
    g_param_spec_boolean ("dark", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_SELECTED] =
    g_param_spec_boolean ("selected", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  gtk_widget_class_set_css_name (widget_class, "palettepreview");
  gtk_widget_class_set_layout_manager_type (widget_class, GTK_TYPE_BIN_LAYOUT);
  gtk_widget_class_set_template_from_resource (widget_class, "/dev/itznoel/Bevy/bevy-palette-preview.ui");
  gtk_widget_class_bind_template_child (widget_class, BevyPalettePreview, image);
  gtk_widget_class_bind_template_child (widget_class, BevyPalettePreview, label);

  g_type_ensure (BEVY_TYPE_PALETTE);
  g_type_ensure (BEVY_TYPE_PALETTE_PREVIEW_COLOR);
}

static void
bevy_palette_preview_init (BevyPalettePreview *self)
{
  gtk_widget_init_template (GTK_WIDGET (self));

  gtk_widget_set_has_tooltip (GTK_WIDGET (self), TRUE);
}

GtkWidget *
bevy_palette_preview_new (BevyPalette *palette)
{
  g_return_val_if_fail (BEVY_IS_PALETTE (palette), NULL);

  return g_object_new (BEVY_TYPE_PALETTE_PREVIEW,
                       "palette", palette,
                       NULL);
}

gboolean
bevy_palette_preview_get_dark (BevyPalettePreview *self)
{
  g_return_val_if_fail (BEVY_IS_PALETTE_PREVIEW (self), FALSE);

  return self->dark;
}

void
bevy_palette_preview_set_dark (BevyPalettePreview *self,
                                 gboolean              dark)
{
  g_return_if_fail (BEVY_IS_PALETTE_PREVIEW (self));

  dark = !!dark;

  if (dark != self->dark)
    {
      self->dark = dark;
      bevy_palette_preview_update_label (self);
      gtk_widget_queue_draw (GTK_WIDGET (self));
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DARK]);
    }
}

BevyPalette *
bevy_palette_preview_get_palette (BevyPalettePreview *self)
{
  g_return_val_if_fail (BEVY_IS_PALETTE_PREVIEW (self), NULL);

  return self->palette;
}

const PangoFontDescription *
bevy_palette_preview_get_font_desc (BevyPalettePreview *self)
{
  g_return_val_if_fail (BEVY_IS_PALETTE_PREVIEW (self), NULL);

  return self->font_desc;
}

void
bevy_palette_preview_set_font_desc (BevyPalettePreview       *self,
                                      const PangoFontDescription *font_desc)
{
  g_return_if_fail (BEVY_IS_PALETTE_PREVIEW (self));

  if (font_desc == self->font_desc)
    return;

  g_clear_pointer (&self->font_desc, pango_font_description_free);
  self->font_desc = pango_font_description_copy (font_desc);
  bevy_palette_preview_update_label (self);
  g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_FONT_DESC]);
}
