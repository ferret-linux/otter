/*
 * bevy-window-dressing.c
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




#include "bevy-application.h"
#include "bevy-window-dressing.h"

struct _BevyWindowDressing
{
  GObject         parent_instance;
  GWeakRef        window_wr;
  BevyPalette  *palette;
  GtkCssProvider *css_provider;
  char           *css_class;
  double          opacity;
  guint           queued_update;
  guint           main_contents : 1;
};

enum {
  PROP_0,
  PROP_OPACITY,
  PROP_PALETTE,
  PROP_WINDOW,
  PROP_MAIN_CONTENTS,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevyWindowDressing, bevy_window_dressing, G_TYPE_OBJECT)

static GParamSpec *properties[N_PROPS];
static guint last_sequence;

static void
bevy_window_dressing_append_surface_rules (BevyWindowDressing *self,
                                            GString             *string,
                                            const BevyPaletteFace *face,
                                            gboolean             dark,
                                            double               window_alpha)
{
  g_autofree char *bg = NULL;
  g_autofree char *fg = NULL;
  g_autofree char *titlebar_bg = NULL;
  g_autofree char *titlebar_fg = NULL;
  g_autofree char *accent_mix_str = NULL;
  g_autofree char *view_bg = NULL;
  g_autofree char *card_bg = NULL;
  g_autofree char *card_shade = NULL;
  g_autofree char *dialog_bg = NULL;
  g_autofree char *popover_shade = NULL;
  g_autofree char *sidebar_bg = NULL;
  g_autofree char *sidebar_backdrop = NULL;
  g_autofree char *sidebar_border = NULL;
  g_autofree char *sidebar_shade = NULL;
  g_autofree char *headerbar_backdrop = NULL;
  g_autofree char *headerbar_border = NULL;
  g_autofree char *headerbar_shade = NULL;
  g_autofree char *headerbar_darker = NULL;
  g_autofree char *backdrop_fg = NULL;
  g_autofree char *backdrop_bg = NULL;
  char window_alpha_str[G_ASCII_DTOSTR_BUF_SIZE];
  char popover_alpha_str[G_ASCII_DTOSTR_BUF_SIZE];
  double popover_alpha;
  GdkRGBA accent_mix;

  g_assert (BEVY_IS_WINDOW_DRESSING (self));

  bg = gdk_rgba_to_string (&face->background);
  fg = gdk_rgba_to_string (&face->foreground);
  titlebar_bg = gdk_rgba_to_string (&face->titlebar_background);
  titlebar_fg = gdk_rgba_to_string (&face->titlebar_foreground);

  window_alpha = CLAMP (window_alpha, 0, 1);
  popover_alpha = MAX (window_alpha, 0.85);

  g_ascii_dtostr (window_alpha_str, sizeof window_alpha_str, window_alpha);
  g_ascii_dtostr (popover_alpha_str, sizeof popover_alpha_str, popover_alpha);

  view_bg = g_strdup_printf ("mix(%s,%s,.04)", bg, fg);
  card_bg = g_strdup_printf ("mix(%s,%s,.08)", bg, fg);
  card_shade = g_strdup_printf ("mix(%s,%s,.15)", bg, fg);
  dialog_bg = g_strdup_printf ("mix(%s,%s,.06)", bg, fg);
  popover_shade = g_strdup_printf ("mix(%s,%s,.15)", bg, fg);
  sidebar_bg = g_strdup_printf ("mix(%s,%s,.04)", bg, fg);
  sidebar_backdrop = g_strdup_printf ("mix(%s,%s,.98)", bg, fg);
  sidebar_border = g_strdup_printf ("mix(%s,%s,.15)", bg, fg);
  sidebar_shade = g_strdup_printf ("mix(%s,%s,.1)", bg, fg);
  headerbar_backdrop = g_strdup_printf ("mix(%s,%s,.985)", titlebar_bg, titlebar_fg);
  headerbar_border = g_strdup_printf ("mix(%s,%s,.15)", titlebar_bg, titlebar_fg);
  headerbar_shade = g_strdup_printf ("mix(%s,%s,.1)", titlebar_bg, titlebar_fg);
  headerbar_darker = g_strdup_printf ("mix(%s,%s,.25)", titlebar_bg, titlebar_fg);
  backdrop_fg = g_strdup_printf ("mix(%s,%s,.025)", fg, bg);
  backdrop_bg = g_strdup_printf ("mix(%s,%s,.985)", fg, bg);

  g_string_append_printf (string,
                          "window.%s { color: %s; background-color: alpha(%s, %s); }\n",
                          self->css_class, fg, bg, window_alpha_str);
  g_string_append_printf (string,
                          "window.%s.fullscreen { background-color: %s; }\n",
                          self->css_class, bg);

  if (!self->main_contents)
    g_string_append_printf (string,
                            "window.%s:backdrop { color: %s; background-color: %s; }\n",
                            self->css_class, backdrop_fg, backdrop_bg);

  g_string_append_printf (string,
                          "window.%s popover > contents { color: %s; background-color: alpha(%s, %s); }\n"
                          "window.%s popover > arrow { background-color: alpha(%s, %s); }\n",
                          self->css_class, titlebar_fg, titlebar_bg, popover_alpha_str,
                          self->css_class, titlebar_bg, popover_alpha_str);

  /* Override the libadwaita CSS custom properties that all its widgets are
   * built from. These cascade down the widget tree, so dialogs, toasts and
   * other surfaces presented within the window are themed uniformly too.
   */
  g_string_append_printf (string,
                          "window.%s {\n"
                          "  --window-bg-color: %s;\n"
                          "  --window-fg-color: %s;\n"
                          "  --view-bg-color: %s;\n"
                          "  --view-fg-color: %s;\n"
                          "  --headerbar-bg-color: %s;\n"
                          "  --headerbar-fg-color: %s;\n"
                          "  --headerbar-backdrop-color: %s;\n"
                          "  --headerbar-border-color: %s;\n"
                          "  --headerbar-shade-color: %s;\n"
                          "  --headerbar-darker-shade-color: %s;\n"
                          "  --card-bg-color: %s;\n"
                          "  --card-fg-color: %s;\n"
                          "  --card-shade-color: %s;\n"
                          "  --dialog-bg-color: %s;\n"
                          "  --dialog-fg-color: %s;\n"
                          "  --popover-bg-color: %s;\n"
                          "  --popover-fg-color: %s;\n"
                          "  --popover-shade-color: %s;\n"
                          "  --sidebar-bg-color: %s;\n"
                          "  --sidebar-fg-color: %s;\n"
                          "  --sidebar-backdrop-color: %s;\n"
                          "  --sidebar-border-color: %s;\n"
                          "  --sidebar-shade-color: %s;\n"
                          "}\n",
                          self->css_class,
                          bg,
                          fg,
                          view_bg,
                          fg,
                          titlebar_bg,
                          titlebar_fg,
                          headerbar_backdrop,
                          headerbar_border,
                          headerbar_shade,
                          headerbar_darker,
                          card_bg,
                          fg,
                          card_shade,
                          dialog_bg,
                          fg,
                          titlebar_bg,
                          titlebar_fg,
                          popover_shade,
                          sidebar_bg,
                          fg,
                          sidebar_backdrop,
                          sidebar_border,
                          sidebar_shade);

  if (!bevy_palette_use_system_accent (self->palette))
    {
      accent_mix = face->indexed[4];
      accent_mix_str = gdk_rgba_to_string (&accent_mix);

      g_string_append_printf (string,
                              "window.%s { --accent-fg-color: %s; --accent-bg-color: mix(%s,%s,.15); }\n",
                              self->css_class,
                              dark ? titlebar_fg : titlebar_bg,
                              accent_mix_str,
                              bg);
    }
}

static void
bevy_window_dressing_update (BevyWindowDressing *self)
{
  g_autoptr(GString) string = NULL;

  g_assert (BEVY_IS_WINDOW_DRESSING (self));

  string = g_string_new (NULL);

  if (self->palette != NULL)
    {
      BevySettings *settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);
      AdwStyleManager *style_manager = adw_style_manager_get_default ();
      gboolean dark = adw_style_manager_get_dark (style_manager);
      const BevyPaletteFace *face = bevy_palette_get_face (self->palette, dark);
      g_autofree char *bg = NULL;
      g_autofree char *fg = NULL;
      g_autofree char *titlebar_bg = NULL;
      g_autofree char *titlebar_fg = NULL;
      g_autofree char *su_fg = NULL;
      g_autofree char *su_bg = NULL;
      g_autofree char *rm_fg = NULL;
      g_autofree char *rm_bg = NULL;
      g_autofree char *bell_fg = NULL;
      g_autofree char *bell_bg = NULL;
      g_autofree char *revealer_bg = NULL;
      char popover_alpha_str[G_ASCII_DTOSTR_BUF_SIZE];
      gboolean visual_process_leader;
      double popover_alpha;

      /* Force clear any background applied to terminals from distro,
       * theme, or user settings so we can be sure our palettes work.
       *
       * See #241 for details
       */
      g_string_append (string, "vte-terminal { background: none; }\n");

      bg = gdk_rgba_to_string (&face->background);
      fg = gdk_rgba_to_string (&face->foreground);
      titlebar_bg = gdk_rgba_to_string (&face->titlebar_background);
      titlebar_fg = gdk_rgba_to_string (&face->titlebar_foreground);
      rm_fg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_REMOTE].foreground);
      rm_bg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_REMOTE].background);
      su_fg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_SUPERUSER].foreground);
      su_bg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_SUPERUSER].background);
      bell_fg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_VISUAL_BELL].foreground);
      bell_bg = gdk_rgba_to_string (&face->scarves[BEVY_PALETTE_SCARF_VISUAL_BELL].background);

      bevy_window_dressing_append_surface_rules (self, string, face, dark, self->opacity);

      if (self->main_contents)
        {
          popover_alpha = MAX (self->opacity, 0.85);
          g_ascii_dtostr (popover_alpha_str, sizeof popover_alpha_str, popover_alpha);
          revealer_bg = g_strdup_printf ("alpha(mix(%s,%s,.05),%s)", titlebar_bg, titlebar_fg, popover_alpha_str);

          g_string_append_printf (string,
                                  "window.%s .window-contents vte-terminal > revealer.size label { color: %s; background-color: %s; }\n",
                                  self->css_class, titlebar_fg, revealer_bg);
          /* It would be super if we could make these match the color of the
           * actual tab contents rather than the active tab profile.
           */
          g_string_append_printf (string,
                                  "window.%s .window-contents toolbarview.overview overlay.card { background-color: %s; color: %s; }\n",
                                  self->css_class, bg, fg);
          g_string_append_printf (string,
                                  "window.%s .window-contents toolbarview.overview tabthumbnail .icon-title-box { color: %s; }\n",
                                  self->css_class, fg);
          g_string_append_printf (string,
                                  "window.%s .window-contents toolbarview.overview { background-color: %s; color: %s; }\n",
                                  self->css_class, titlebar_bg, titlebar_fg);
          g_string_append_printf (string,
                                  "window.%s .window-contents revealer.raised.top-bar { background-color: %s; color: %s; }\n",
                                  self->css_class, titlebar_bg, titlebar_fg);
          g_string_append_printf (string,
                                  "window.%s .window-contents box.visual-bell headerbar { background-color: transparent; }\n"
                                  "window.%s .window-contents box.visual-bell { animation: visual-bell-%s-%s 0.3s ease-out; }\n"
                                  "@keyframes visual-bell-%s-%s { 50%% { background-color: %s; color: %s; } }\n",
                                  self->css_class,
                                  self->css_class, self->css_class, dark ? "dark" : "light",
                                  self->css_class, dark ? "dark" : "light", bell_bg, bell_fg);
          g_string_append_printf (string,
                                  "window.%s .window-contents banner > revealer > widget { background-color: %s; color: %s; }\n",
                                  self->css_class, bell_bg, bell_fg);

          g_string_append_printf (string,
                                  "window.%s taboverview.window-contents tabthumbnail .tab-close-button image { background-color: alpha(%s,.15); color: %s; }\n"
                                  "window.%s taboverview.window-contents tabthumbnail .tab-close-button:hover image { background-color: alpha(%s,.25); }\n"
                                  "window.%s taboverview.window-contents tabthumbnail .tab-close-button:active image { background-color: alpha(%s,.55); }\n",
                                  self->css_class, fg, fg,
                                  self->css_class, fg,
                                  self->css_class, fg);

          visual_process_leader = bevy_settings_get_visual_process_leader (settings);

          g_string_append_printf (string,
                                  "window.%s .window-contents > revealer windowhandle { color: %s; background-color: %s; }\n",
                                  self->css_class, titlebar_fg, titlebar_bg);
          g_string_append_printf (string,
                                  "window.%s:backdrop .window-contents revealer > windowhandle { color: mix(%s,%s,.025); background-color: mix(%s,%s,.99); }\n",
                                  self->css_class, fg, bg, fg, bg);

          if (visual_process_leader)
            {
              g_string_append_printf (string,
                                      "window.%s.remote .window-contents headerbar { background-color: %s; color: %s; }\n"
                                      "window.%s.remote .window-contents toolbarview > revealer > windowhandle { background-color: %s; color: %s; }\n",
                                      self->css_class, rm_bg, rm_fg,
                                      self->css_class, rm_bg, rm_fg);
              g_string_append_printf (string,
                                      "window.%s.superuser .window-contents headerbar { background-color: %s; color: %s; }\n"
                                      "window.%s.superuser .window-contents toolbarview > revealer > windowhandle { background-color: %s; color: %s; }\n",
                                      self->css_class, su_bg, su_fg,
                                      self->css_class, su_bg, su_fg);
            }
        }
    }

  gtk_css_provider_load_from_string (self->css_provider, string->str);
}

static gboolean
bevy_window_dressing_update_idle (gpointer user_data)
{
  BevyWindowDressing *self = BEVY_WINDOW_DRESSING (user_data);

  self->queued_update = 0;

  bevy_window_dressing_update (self);

  return G_SOURCE_REMOVE;
}

static void
bevy_window_dressing_queue_update (BevyWindowDressing *self)
{
  g_assert (BEVY_IS_WINDOW_DRESSING (self));

  if (self->queued_update == 0)
    self->queued_update = g_idle_add_full (G_PRIORITY_HIGH_IDLE,
                                           bevy_window_dressing_update_idle,
                                           self, NULL);
}

static void
bevy_window_dressing_set_window (BevyWindowDressing *self,
                                 GtkWidget          *root)
{
  g_assert (BEVY_IS_WINDOW_DRESSING (self));
  g_assert (GTK_IS_WIDGET (root));

  g_weak_ref_set (&self->window_wr, root);

  gtk_widget_add_css_class (root, self->css_class);
}

static void
bevy_window_dressing_constructed (GObject *object)
{
  BevyWindowDressing *self = (BevyWindowDressing *)object;
  AdwStyleManager *style_manager = adw_style_manager_get_default ();
  BevySettings *settings = bevy_application_get_settings (BEVY_APPLICATION_DEFAULT);

  G_OBJECT_CLASS (bevy_window_dressing_parent_class)->constructed (object);

  g_signal_connect_object (settings,
                           "notify::visual-process-leader",
                           G_CALLBACK (bevy_window_dressing_queue_update),
                           self,
                           G_CONNECT_SWAPPED);

  gtk_style_context_add_provider_for_display (gdk_display_get_default (),
                                              GTK_STYLE_PROVIDER (self->css_provider),
                                              GTK_STYLE_PROVIDER_PRIORITY_USER + 1);

  g_signal_connect_object (style_manager,
                           "notify::dark",
                           G_CALLBACK (bevy_window_dressing_queue_update),
                           self,
                           G_CONNECT_SWAPPED);
  g_signal_connect_object (style_manager,
                           "notify::color-scheme",
                           G_CALLBACK (bevy_window_dressing_queue_update),
                           self,
                           G_CONNECT_SWAPPED);

  bevy_window_dressing_queue_update (self);
}

static void
bevy_window_dressing_dispose (GObject *object)
{
  BevyWindowDressing *self = (BevyWindowDressing *)object;

  bevy_window_dressing_set_palette (self, NULL);

  if (self->css_provider != NULL) {
    gtk_style_context_remove_provider_for_display (gdk_display_get_default (),
                                                   GTK_STYLE_PROVIDER (self->css_provider));
    g_clear_object (&self->css_provider);
  }

  g_clear_object (&self->palette);
  g_weak_ref_set (&self->window_wr, NULL);

  g_clear_handle_id (&self->queued_update, g_source_remove);

  G_OBJECT_CLASS (bevy_window_dressing_parent_class)->dispose (object);
}

static void
bevy_window_dressing_finalize (GObject *object)
{
  BevyWindowDressing *self = (BevyWindowDressing *)object;

  g_weak_ref_clear (&self->window_wr);
  g_clear_pointer (&self->css_class, g_free);

  g_assert (self->palette == NULL);
  g_assert (self->queued_update == 0);
  g_assert (self->css_provider == NULL);

  G_OBJECT_CLASS (bevy_window_dressing_parent_class)->finalize (object);
}

static void
bevy_window_dressing_get_property (GObject    *object,
                                     guint       prop_id,
                                     GValue     *value,
                                     GParamSpec *pspec)
{
  BevyWindowDressing *self = BEVY_WINDOW_DRESSING (object);

  switch (prop_id)
    {
    case PROP_OPACITY:
      g_value_set_double (value, bevy_window_dressing_get_opacity (self));
      break;

    case PROP_PALETTE:
      g_value_set_object (value, bevy_window_dressing_get_palette (self));
      break;

    case PROP_WINDOW:
      g_value_take_object (value, bevy_window_dressing_dup_window (self));
      break;

    case PROP_MAIN_CONTENTS:
      g_value_set_boolean (value, self->main_contents);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_window_dressing_set_property (GObject      *object,
                                     guint         prop_id,
                                     const GValue *value,
                                     GParamSpec   *pspec)
{
  BevyWindowDressing *self = BEVY_WINDOW_DRESSING (object);

  switch (prop_id)
    {
    case PROP_OPACITY:
      bevy_window_dressing_set_opacity (self, g_value_get_double (value));
      break;

    case PROP_PALETTE:
      bevy_window_dressing_set_palette (self, g_value_get_object (value));
      break;

    case PROP_WINDOW:
      bevy_window_dressing_set_window (self, g_value_get_object (value));
      break;

    case PROP_MAIN_CONTENTS:
      self->main_contents = g_value_get_boolean (value);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_window_dressing_class_init (BevyWindowDressingClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->constructed = bevy_window_dressing_constructed;
  object_class->dispose = bevy_window_dressing_dispose;
  object_class->finalize = bevy_window_dressing_finalize;
  object_class->get_property = bevy_window_dressing_get_property;
  object_class->set_property = bevy_window_dressing_set_property;

  properties[PROP_OPACITY] =
    g_param_spec_double ("opacity", NULL, NULL,
                         0, 1, 1,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_PALETTE] =
    g_param_spec_object ("palette", NULL, NULL,
                         BEVY_TYPE_PALETTE,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_WINDOW] =
    g_param_spec_object ("window", NULL, NULL,
                         GTK_TYPE_WIDGET,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_MAIN_CONTENTS] =
    g_param_spec_boolean ("main-contents", NULL, NULL,
                          TRUE,
                          (G_PARAM_READWRITE |
                           G_PARAM_CONSTRUCT_ONLY |
                           G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);
}

static void
bevy_window_dressing_init (BevyWindowDressing *self)
{
  self->css_provider = gtk_css_provider_new ();
  self->css_class = g_strdup_printf ("window-dressing-%u", ++last_sequence);
  self->opacity = 1.0;
  self->main_contents = TRUE;

  g_weak_ref_init (&self->window_wr, NULL);
}

GtkWidget *
bevy_window_dressing_dup_window (BevyWindowDressing *self)
{
  g_return_val_if_fail (BEVY_IS_WINDOW_DRESSING (self), NULL);

  return g_weak_ref_get (&self->window_wr);
}

BevyWindowDressing *
bevy_window_dressing_new (BevyWindow *window)
{
  g_return_val_if_fail (BEVY_IS_WINDOW (window), NULL);

  return g_object_new (BEVY_TYPE_WINDOW_DRESSING,
                       "window", window,
                       NULL);
}

BevyWindowDressing *
bevy_window_dressing_new_for_root (GtkWidget *root,
                                   gboolean   main_contents)
{
  g_return_val_if_fail (GTK_IS_WIDGET (root), NULL);

  return g_object_new (BEVY_TYPE_WINDOW_DRESSING,
                       "window", root,
                       "main-contents", main_contents,
                       NULL);
}

BevyPalette *
bevy_window_dressing_get_palette (BevyWindowDressing *self)
{
  g_return_val_if_fail (BEVY_IS_WINDOW_DRESSING (self), NULL);

  return self->palette;
}

void
bevy_window_dressing_set_palette (BevyWindowDressing *self,
                                    BevyPalette        *palette)
{
  g_return_if_fail (BEVY_IS_WINDOW_DRESSING (self));
  g_return_if_fail (!palette || BEVY_IS_PALETTE (palette));

  if (g_set_object (&self->palette, palette))
    {
      bevy_window_dressing_queue_update (self);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PALETTE]);
    }
}

double
bevy_window_dressing_get_opacity (BevyWindowDressing *self)
{
  g_return_val_if_fail (BEVY_IS_WINDOW_DRESSING (self), 1.);

  return self->opacity;
}

void
bevy_window_dressing_set_opacity (BevyWindowDressing *self,
                                    double                opacity)
{
  g_return_if_fail (BEVY_IS_WINDOW_DRESSING (self));

  if (opacity != self->opacity)
    {
      self->opacity = opacity;
      bevy_window_dressing_queue_update (self);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_OPACITY]);
    }
}
