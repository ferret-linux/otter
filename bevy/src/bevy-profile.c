/*
 * bevy-profile.c
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

#include <gio/gio.h>

#include "bevy-application.h"
#include "bevy-enums.h"
#include "bevy-profile.h"
#include "bevy-util.h"

struct _BevyProfile
{
  GObject parent_instance;
  GSettings *settings;
  char *uuid;
  GListStore *custom_links;
};

enum {
  PROP_0,
  PROP_BACKSPACE_BINDING,
  PROP_BOLD_IS_BRIGHT,
  PROP_CELL_HEIGHT_SCALE,
  PROP_CELL_WIDTH_SCALE,
  PROP_CJK_AMBIGUOUS_WIDTH,
  PROP_CUSTOM_COMMAND,
  PROP_DEFAULT_CONTAINER,
  PROP_DELETE_BINDING,
  PROP_EXIT_ACTION,
  PROP_LABEL,
  PROP_LIMIT_SCROLLBACK,
  PROP_LOGIN_SHELL,
  PROP_MARGIN_X,
  PROP_MARGIN_Y,
  PROP_OPACITY,
  PROP_PALETTE,
  PROP_PALETTE_ID,
  PROP_PRESERVE_CONTAINER,
  PROP_PRESERVE_DIRECTORY,
  PROP_SCROLL_ON_KEYSTROKE,
  PROP_SCROLL_ON_OUTPUT,
  PROP_SCROLLBACK_LINES,
  PROP_USE_CUSTOM_COMMAND,
  PROP_USE_PROXY,
  PROP_UUID,
  PROP_CUSTOM_LINKS,
  N_PROPS
};

enum {
  CUSTOM_LINKS_CHANGED,
  N_SIGNALS
};

G_DEFINE_FINAL_TYPE (BevyProfile, bevy_profile, G_TYPE_OBJECT)

static GParamSpec *properties [N_PROPS];
static guint signals[N_SIGNALS];

static void
read_custom_links_from_settings (BevyProfile *self)
{
  g_autoptr(GListModel) custom_links = NULL;
  g_autoptr(GVariant) variant = NULL;
  const char *pattern, *target;
  GVariantIter iter;
  gboolean changed = FALSE;

  g_assert (BEVY_IS_PROFILE (self));

  custom_links = bevy_profile_list_custom_links (self);
  variant = g_settings_get_value (self->settings, BEVY_PROFILE_KEY_CUSTOM_LINKS);

  if (variant == NULL)
    {
      g_warning ("Invalid variant type for custom-links: variant is null");
      return;
    }

  if (!g_variant_is_of_type (variant, G_VARIANT_TYPE ("a(ss)")))
    {
      g_warning ("Invalid variant type for custom-links: %s",
                g_variant_get_type_string (variant));
      return;
    }

  /* Check if there are actual changes */
  g_variant_iter_init (&iter, variant);

  changed = g_variant_iter_n_children (&iter) != (gsize)g_list_model_get_n_items (custom_links);

  if (!changed)
    {
      guint index = 0;

      while (g_variant_iter_next (&iter, "(&s&s)", &pattern, &target))
        {
          g_autoptr(BevyCustomLink) custom_link = g_list_model_get_item (custom_links, index);
          g_autofree char *this_pattern = bevy_custom_link_dup_pattern (custom_link);
          g_autofree char *this_target = bevy_custom_link_dup_target (custom_link);

          if (g_strcmp0 (this_pattern, pattern) != 0 ||
              g_strcmp0 (this_target, target) != 0)
            {
              changed = TRUE;
              break;
            }

          index++;
        }

      g_variant_iter_init (&iter, variant);
    }

  /* If there are changes, regenerate all custom link objects */
  if (changed)
    {
      g_list_store_remove_all (self->custom_links);

      while (g_variant_iter_next (&iter, "(&s&s)", &pattern, &target))
        {
          BevyCustomLink *custom_link = bevy_custom_link_new_with_strings (pattern, target);

          g_list_store_append (self->custom_links, custom_link);
        }
    }

  /* We emit the signal in all cases, because we might be the origin of the update
     and already have the correct copy of custom links
  */
  g_debug ("Custom links have changed, emitting signal (changed = %d)", changed);
  g_signal_emit (self, signals[CUSTOM_LINKS_CHANGED], 0);
}

static void
write_custom_links_to_settings (BevyProfile *self)
{
  g_autoptr(GVariantBuilder) builder = NULL;
  g_autoptr(GListModel) custom_links = NULL;

  g_assert (BEVY_IS_PROFILE (self));

  builder = g_variant_builder_new (G_VARIANT_TYPE ("a(ss)"));
  custom_links = bevy_profile_list_custom_links (self);

  for (guint i = 0, len = g_list_model_get_n_items (custom_links); i < len; i++)
    {
      g_autoptr(BevyCustomLink) link = g_list_model_get_item (custom_links, i);
      g_autofree char *pattern = bevy_custom_link_dup_pattern (link);
      g_autofree char *target = bevy_custom_link_dup_target (link);

      g_variant_builder_add (builder, "(ss)", pattern, target);
    }

  g_settings_set_value (self->settings,
                        BEVY_PROFILE_KEY_CUSTOM_LINKS,
                        g_variant_new ("a(ss)", builder));
}


static void
bevy_profile_changed_cb (BevyProfile *self,
                           const char    *key,
                           GSettings     *settings)
{
  g_assert (BEVY_IS_PROFILE (self));
  g_assert (key != NULL);
  g_assert (G_IS_SETTINGS (settings));

  if (FALSE) {}
  else if (g_str_equal (key, BEVY_PROFILE_KEY_LABEL))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_LABEL]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_DEFAULT_CONTAINER))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DEFAULT_CONTAINER]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_EXIT_ACTION))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_EXIT_ACTION]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_PALETTE))
    {
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PALETTE]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PALETTE_ID]);
    }
  else if (g_str_equal (key, BEVY_PROFILE_KEY_OPACITY))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_OPACITY]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_LIMIT_SCROLLBACK))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_LIMIT_SCROLLBACK]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_SCROLLBACK_LINES))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_SCROLLBACK_LINES]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_BACKSPACE_BINDING))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_BACKSPACE_BINDING]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_DELETE_BINDING))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DELETE_BINDING]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_CJK_AMBIGUOUS_WIDTH))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CJK_AMBIGUOUS_WIDTH]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_BOLD_IS_BRIGHT))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_BOLD_IS_BRIGHT]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_CELL_HEIGHT_SCALE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CELL_HEIGHT_SCALE]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_CELL_WIDTH_SCALE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CELL_WIDTH_SCALE]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_LOGIN_SHELL))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_LOGIN_SHELL]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_MARGIN_X))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_MARGIN_X]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_MARGIN_Y))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_MARGIN_Y]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_CUSTOM_COMMAND))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CUSTOM_COMMAND]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_USE_CUSTOM_COMMAND))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_USE_CUSTOM_COMMAND]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_USE_PROXY))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_USE_PROXY]);
  else if (g_str_equal (key, BEVY_PROFILE_KEY_CUSTOM_LINKS))
    read_custom_links_from_settings (self);
}

static void
bevy_profile_constructed (GObject *object)
{
  BevyProfile *self = (BevyProfile *)object;
  g_autofree char *path = NULL;

  G_OBJECT_CLASS (bevy_profile_parent_class)->constructed (object);

  if (self->uuid == NULL)
    self->uuid = g_dbus_generate_guid ();

  self->custom_links = g_list_store_new (BEVY_TYPE_CUSTOM_LINK);

  path = g_strdup_printf (APP_SCHEMA_PATH"Profiles/%s/", self->uuid);
  self->settings = g_settings_new_with_path (APP_SCHEMA_PROFILE_ID, path);

  g_signal_connect_object (self->settings,
                           "changed",
                           G_CALLBACK (bevy_profile_changed_cb),
                           self,
                           G_CONNECT_SWAPPED);

  read_custom_links_from_settings (self);
}

static void
bevy_profile_dispose (GObject *object)
{
  BevyProfile *self = (BevyProfile *)object;

  g_clear_object (&self->settings);
  g_clear_object (&self->custom_links);
  g_clear_pointer (&self->uuid, g_free);

  G_OBJECT_CLASS (bevy_profile_parent_class)->dispose (object);
}

static void
bevy_profile_get_property (GObject    *object,
                             guint       prop_id,
                             GValue     *value,
                             GParamSpec *pspec)
{
  BevyProfile *self = BEVY_PROFILE (object);

  switch (prop_id)
    {
    case PROP_CJK_AMBIGUOUS_WIDTH:
      g_value_set_enum (value, bevy_profile_get_cjk_ambiguous_width (self));
      break;

    case PROP_BACKSPACE_BINDING:
      g_value_set_enum (value, bevy_profile_get_backspace_binding (self));
      break;

    case PROP_BOLD_IS_BRIGHT:
      g_value_set_boolean (value, bevy_profile_get_bold_is_bright (self));
      break;

    case PROP_CELL_HEIGHT_SCALE:
      g_value_set_double (value, bevy_profile_get_cell_height_scale (self));
      break;

    case PROP_CELL_WIDTH_SCALE:
      g_value_set_double (value, bevy_profile_get_cell_width_scale (self));
      break;

    case PROP_CUSTOM_COMMAND:
      g_value_take_string (value, bevy_profile_dup_custom_command (self));
      break;

    case PROP_DEFAULT_CONTAINER:
      g_value_take_string (value, bevy_profile_dup_default_container (self));
      break;

    case PROP_DELETE_BINDING:
      g_value_set_enum (value, bevy_profile_get_delete_binding (self));
      break;

    case PROP_EXIT_ACTION:
      g_value_set_enum (value, bevy_profile_get_exit_action (self));
      break;

    case PROP_LABEL:
      g_value_take_string (value, bevy_profile_dup_label (self));
      break;

    case PROP_LIMIT_SCROLLBACK:
      g_value_set_boolean (value, bevy_profile_get_limit_scrollback (self));
      break;

    case PROP_LOGIN_SHELL:
      g_value_set_boolean (value, bevy_profile_get_login_shell (self));
      break;

    case PROP_MARGIN_X:
      g_value_set_int (value, bevy_profile_get_margin_x (self));
      break;

    case PROP_MARGIN_Y:
      g_value_set_int (value, bevy_profile_get_margin_y (self));
      break;

    case PROP_OPACITY:
      g_value_set_double (value, bevy_profile_get_opacity (self));
      break;

    case PROP_PALETTE:
      g_value_take_object (value, bevy_profile_dup_palette (self));
      break;

    case PROP_PALETTE_ID:
      g_value_take_string (value, bevy_profile_dup_palette_id (self));
      break;

    case PROP_PRESERVE_CONTAINER:
      g_value_set_enum (value, bevy_profile_get_preserve_container (self));
      break;

    case PROP_PRESERVE_DIRECTORY:
      g_value_set_enum (value, bevy_profile_get_preserve_directory (self));
      break;

    case PROP_SCROLL_ON_KEYSTROKE:
      g_value_set_boolean (value, bevy_profile_get_scroll_on_keystroke (self));
      break;

    case PROP_SCROLL_ON_OUTPUT:
      g_value_set_boolean (value, bevy_profile_get_scroll_on_output (self));
      break;

    case PROP_SCROLLBACK_LINES:
      g_value_set_int (value, bevy_profile_get_scrollback_lines (self));
      break;

    case PROP_USE_CUSTOM_COMMAND:
      g_value_set_boolean (value, bevy_profile_get_use_custom_command (self));
      break;

    case PROP_USE_PROXY:
      g_value_set_boolean (value, bevy_profile_get_use_proxy (self));
      break;

    case PROP_UUID:
      g_value_set_string (value, bevy_profile_get_uuid (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_profile_set_property (GObject      *object,
                             guint         prop_id,
                             const GValue *value,
                             GParamSpec   *pspec)
{
  BevyProfile *self = BEVY_PROFILE (object);

  switch (prop_id)
    {
    case PROP_CJK_AMBIGUOUS_WIDTH:
      bevy_profile_set_cjk_ambiguous_width (self, g_value_get_enum (value));
      break;

    case PROP_BACKSPACE_BINDING:
      bevy_profile_set_backspace_binding (self, g_value_get_enum (value));
      break;

    case PROP_BOLD_IS_BRIGHT:
      bevy_profile_set_bold_is_bright (self, g_value_get_boolean (value));
      break;

    case PROP_CELL_HEIGHT_SCALE:
      bevy_profile_set_cell_height_scale (self, g_value_get_double (value));
      break;

    case PROP_CELL_WIDTH_SCALE:
      bevy_profile_set_cell_width_scale (self, g_value_get_double (value));
      break;

    case PROP_CUSTOM_COMMAND:
      bevy_profile_set_custom_command (self, g_value_get_string (value));
      break;

    case PROP_DEFAULT_CONTAINER:
      bevy_profile_set_default_container (self, g_value_get_string (value));
      break;

    case PROP_DELETE_BINDING:
      bevy_profile_set_delete_binding (self, g_value_get_enum (value));
      break;

    case PROP_EXIT_ACTION:
      bevy_profile_set_exit_action (self, g_value_get_enum (value));
      break;

    case PROP_LABEL:
      bevy_profile_set_label (self, g_value_get_string (value));
      break;

    case PROP_LIMIT_SCROLLBACK:
      bevy_profile_set_limit_scrollback (self, g_value_get_boolean (value));
      break;

    case PROP_LOGIN_SHELL:
      bevy_profile_set_login_shell (self, g_value_get_boolean (value));
      break;

    case PROP_MARGIN_X:
      bevy_profile_set_margin_x (self, g_value_get_int (value));
      break;

    case PROP_MARGIN_Y:
      bevy_profile_set_margin_y (self, g_value_get_int (value));
      break;

    case PROP_OPACITY:
      bevy_profile_set_opacity (self, g_value_get_double (value));
      break;

    case PROP_PALETTE:
      bevy_profile_set_palette (self, g_value_get_object (value));
      break;

    case PROP_PALETTE_ID:
      {
        const char *id = g_value_get_string (value);
        g_autoptr(BevyPalette) palette = bevy_palette_lookup (id);

        if (palette == NULL)
          palette = bevy_palette_lookup ("gnome");

        bevy_profile_set_palette (self, palette);
      }
      break;

    case PROP_PRESERVE_CONTAINER:
      bevy_profile_set_preserve_container (self, g_value_get_enum (value));
      break;

    case PROP_PRESERVE_DIRECTORY:
      bevy_profile_set_preserve_directory (self, g_value_get_enum (value));
      break;

    case PROP_SCROLL_ON_KEYSTROKE:
      bevy_profile_set_scroll_on_keystroke (self, g_value_get_boolean (value));
      break;

    case PROP_SCROLL_ON_OUTPUT:
      bevy_profile_set_scroll_on_output (self, g_value_get_boolean (value));
      break;

    case PROP_SCROLLBACK_LINES:
      bevy_profile_set_scrollback_lines (self, g_value_get_int (value));
      break;

    case PROP_USE_CUSTOM_COMMAND:
      bevy_profile_set_use_custom_command (self, g_value_get_boolean (value));
      break;

    case PROP_USE_PROXY:
      bevy_profile_set_use_proxy (self, g_value_get_boolean (value));
      break;

    case PROP_CUSTOM_LINKS:
      self->custom_links = g_value_get_object (value);
      break;

    case PROP_UUID:
      self->uuid = g_value_dup_string (value);
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_profile_class_init (BevyProfileClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->constructed = bevy_profile_constructed;
  object_class->dispose = bevy_profile_dispose;
  object_class->get_property = bevy_profile_get_property;
  object_class->set_property = bevy_profile_set_property;

  properties[PROP_CJK_AMBIGUOUS_WIDTH] =
    g_param_spec_enum ("cjk-ambiguous-width", NULL, NULL,
                       BEVY_TYPE_CJK_AMBIGUOUS_WIDTH,
                       BEVY_CJK_AMBIGUOUS_WIDTH_NARROW,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_CUSTOM_COMMAND] =
    g_param_spec_string ("custom-command", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_BACKSPACE_BINDING] =
    g_param_spec_enum ("backspace-binding", NULL, NULL,
                       VTE_TYPE_ERASE_BINDING,
                       VTE_ERASE_AUTO,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_BOLD_IS_BRIGHT] =
    g_param_spec_boolean ("bold-is-bright", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_CELL_HEIGHT_SCALE] =
    g_param_spec_double ("cell-height-scale", NULL, NULL,
                         1.0, 2.0, 1.0,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_CELL_WIDTH_SCALE] =
    g_param_spec_double ("cell-width-scale", NULL, NULL,
                         1.0, 2.0, 1.0,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_DEFAULT_CONTAINER] =
    g_param_spec_string ("default-container", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_DELETE_BINDING] =
    g_param_spec_enum ("delete-binding", NULL, NULL,
                       VTE_TYPE_ERASE_BINDING,
                       VTE_ERASE_AUTO,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_EXIT_ACTION] =
    g_param_spec_enum ("exit-action", NULL, NULL,
                       BEVY_TYPE_EXIT_ACTION,
                       BEVY_EXIT_ACTION_CLOSE,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_LABEL] =
    g_param_spec_string ("label", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_LIMIT_SCROLLBACK] =
    g_param_spec_boolean ("limit-scrollback", NULL, NULL,
                         FALSE,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_LOGIN_SHELL] =
    g_param_spec_boolean ("login-shell", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_MARGIN_X] =
    g_param_spec_int ("margin-x", NULL, NULL,
                      0, 100, 6,
                      (G_PARAM_READWRITE |
                       G_PARAM_EXPLICIT_NOTIFY |
                       G_PARAM_STATIC_STRINGS));

  properties[PROP_MARGIN_Y] =
    g_param_spec_int ("margin-y", NULL, NULL,
                      0, 100, 6,
                      (G_PARAM_READWRITE |
                       G_PARAM_EXPLICIT_NOTIFY |
                       G_PARAM_STATIC_STRINGS));

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

  properties[PROP_PALETTE_ID] =
    g_param_spec_string ("palette-id", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_PRESERVE_CONTAINER] =
    g_param_spec_enum ("preserve-container", NULL, NULL,
                       BEVY_TYPE_PRESERVE_CONTAINER,
                       BEVY_PRESERVE_CONTAINER_NEVER,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_PRESERVE_DIRECTORY] =
    g_param_spec_enum ("preserve-directory", NULL, NULL,
                       BEVY_TYPE_PRESERVE_DIRECTORY,
                       BEVY_PRESERVE_DIRECTORY_SAFE,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_SCROLL_ON_KEYSTROKE] =
    g_param_spec_boolean ("scroll-on-keystroke", NULL, NULL,
                         FALSE,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_SCROLL_ON_OUTPUT] =
    g_param_spec_boolean ("scroll-on-output", NULL, NULL,
                         FALSE,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_SCROLLBACK_LINES] =
    g_param_spec_int ("scrollback-lines", NULL, NULL,
                      0, G_MAXINT, 10000,
                      (G_PARAM_READWRITE |
                       G_PARAM_EXPLICIT_NOTIFY |
                       G_PARAM_STATIC_STRINGS));

  properties[PROP_USE_CUSTOM_COMMAND] =
    g_param_spec_boolean ("use-custom-command", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_USE_PROXY] =
    g_param_spec_boolean ("use-proxy", NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_UUID] =
    g_param_spec_string ("uuid", NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_CONSTRUCT_ONLY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_CUSTOM_LINKS] =
    g_param_spec_object ("custom-links", NULL, NULL,
                         G_TYPE_LIST_STORE,
                         (G_PARAM_READABLE |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);

  signals[CUSTOM_LINKS_CHANGED] = g_signal_new ("custom-links-changed",
                                                G_TYPE_FROM_CLASS (klass),
                                                G_SIGNAL_RUN_LAST,
                                                0,
                                                NULL, NULL,
                                                NULL,
                                                G_TYPE_NONE, 0);

}

static void
bevy_profile_init (BevyProfile *self)
{
}

BevyProfile *
bevy_profile_new (const char *uuid)
{
  return g_object_new (BEVY_TYPE_PROFILE,
                       "uuid", uuid,
                       NULL);
}

/**
 * bevy_profile_get_uuid:
 * @self: a #BevyProfile
 *
 * Gets the UUID for the profile.
 *
 * Returns: (not nullable): the profile UUID
 */
const char *
bevy_profile_get_uuid (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return self->uuid;
}

char *
bevy_profile_dup_label (BevyProfile *self)
{
  g_autofree char *label = NULL;

  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  label = g_settings_get_string (self->settings, BEVY_PROFILE_KEY_LABEL);

  if (bevy_str_empty0 (label))
    g_set_str (&label, _("Untitled Profile"));

  return g_steal_pointer (&label);
}

void
bevy_profile_set_label (BevyProfile *self,
                          const char    *label)
{
  g_autoptr(GMenuModel) menu = NULL;

  g_return_if_fail (BEVY_IS_PROFILE (self));

  if (label == NULL)
    label = "";

  g_settings_set_string (self->settings, BEVY_PROFILE_KEY_LABEL, label);
}

gboolean
bevy_profile_get_scroll_on_keystroke (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_SCROLL_ON_KEYSTROKE);
}

void
bevy_profile_set_scroll_on_keystroke (BevyProfile *self,
                                        gboolean       scroll_on_keystroke)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_SCROLL_ON_KEYSTROKE,
                          scroll_on_keystroke);
}

gboolean
bevy_profile_get_scroll_on_output (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_SCROLL_ON_OUTPUT);
}

void
bevy_profile_set_scroll_on_output (BevyProfile *self,
                                     gboolean       scroll_on_output)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_SCROLL_ON_OUTPUT,
                          scroll_on_output);
}

char *
bevy_profile_dup_default_container (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return g_settings_get_string (self->settings, BEVY_PROFILE_KEY_DEFAULT_CONTAINER);
}

void
bevy_profile_set_default_container (BevyProfile *self,
                                      const char    *default_container)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  if (default_container == NULL)
    default_container = "";

  g_settings_set_string (self->settings,
                         BEVY_PROFILE_KEY_DEFAULT_CONTAINER,
                         default_container);
}

BevyExitAction
bevy_profile_get_exit_action (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_EXIT_ACTION);
}

void
bevy_profile_set_exit_action (BevyProfile    *self,
                                BevyExitAction  exit_action)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (exit_action <= BEVY_EXIT_ACTION_CLOSE);

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_EXIT_ACTION,
                       exit_action);
}

BevyPreserveContainer
bevy_profile_get_preserve_container (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_PRESERVE_CONTAINER);
}

void
bevy_profile_set_preserve_container (BevyProfile           *self,
                                       BevyPreserveContainer  preserve_container)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (preserve_container <= BEVY_PRESERVE_CONTAINER_ALWAYS);

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_PRESERVE_CONTAINER,
                       preserve_container);
}

BevyPreserveDirectory
bevy_profile_get_preserve_directory (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_PRESERVE_DIRECTORY);
}

void
bevy_profile_set_preserve_directory (BevyProfile           *self,
                                       BevyPreserveDirectory  preserve_directory)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (preserve_directory <= BEVY_PRESERVE_DIRECTORY_ALWAYS);

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_PRESERVE_DIRECTORY,
                       preserve_directory);
}

BevyProfile *
bevy_profile_duplicate (BevyProfile *self)
{
  g_autoptr(BevyProfile) copy = NULL;
  g_autoptr(GSettingsSchema) schema = NULL;
  g_auto(GStrv) keys = NULL;

  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  g_object_get (self->settings,
                "settings-schema", &schema,
                NULL);

  copy = bevy_profile_new (NULL);
  keys = g_settings_schema_list_keys (schema);

  for (guint i = 0; keys[i]; i++)
    {
      g_autoptr(GVariant) user_value = g_settings_get_user_value (self->settings, keys[i]);

      if (user_value != NULL)
        g_settings_set_value (copy->settings, keys[i], user_value);
    }

  bevy_application_add_profile (BEVY_APPLICATION_DEFAULT, copy);

  return g_steal_pointer (&copy);
}

char *
bevy_profile_dup_palette_id (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return g_settings_get_string (self->settings, BEVY_PROFILE_KEY_PALETTE);
}

BevyPalette *
bevy_profile_dup_palette (BevyProfile *self)
{
  g_autofree char *name = NULL;

  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  name = g_settings_get_string (self->settings, BEVY_PROFILE_KEY_PALETTE);

  return bevy_palette_lookup (name);
}

void
bevy_profile_set_palette (BevyProfile *self,
                            BevyPalette *palette)
{
  const char *id = "gnome";

  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (!palette || BEVY_IS_PALETTE (palette));

  if (palette != NULL)
    id = bevy_palette_get_id (palette);

  g_settings_set_string (self->settings, BEVY_PROFILE_KEY_PALETTE, id);
}

double
bevy_profile_get_opacity (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 1.);

  return g_settings_get_double (self->settings, BEVY_PROFILE_KEY_OPACITY);
}

void
bevy_profile_set_opacity (BevyProfile *self,
                            double         opacity)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  opacity = CLAMP (opacity, 0, 1);

  g_settings_set_double (self->settings,
                         BEVY_PROFILE_KEY_OPACITY,
                         opacity);
}

gboolean
bevy_profile_get_limit_scrollback (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_LIMIT_SCROLLBACK);
}

void
bevy_profile_set_limit_scrollback (BevyProfile *self,
                                     gboolean       limit_scrollback)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_LIMIT_SCROLLBACK,
                          limit_scrollback);
}

int
bevy_profile_get_scrollback_lines (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_int (self->settings, BEVY_PROFILE_KEY_SCROLLBACK_LINES);
}

void
bevy_profile_set_scrollback_lines (BevyProfile *self,
                                     int            scrollback_lines)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_int (self->settings,
                      BEVY_PROFILE_KEY_SCROLLBACK_LINES,
                      scrollback_lines);
}

GSettings *
bevy_profile_dup_settings (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return g_object_ref (self->settings);
}

VteEraseBinding
bevy_profile_get_backspace_binding (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_BACKSPACE_BINDING);
}

void
bevy_profile_set_backspace_binding (BevyProfile   *self,
                                      VteEraseBinding  backspace_binding)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_BACKSPACE_BINDING,
                       backspace_binding);
}

VteEraseBinding
bevy_profile_get_delete_binding (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_DELETE_BINDING);
}

void
bevy_profile_set_delete_binding (BevyProfile   *self,
                                   VteEraseBinding  delete_binding)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_DELETE_BINDING,
                       delete_binding);
}

BevyCjkAmbiguousWidth
bevy_profile_get_cjk_ambiguous_width (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 1);

  return g_settings_get_enum (self->settings, BEVY_PROFILE_KEY_CJK_AMBIGUOUS_WIDTH);
}

void
bevy_profile_set_cjk_ambiguous_width (BevyProfile           *self,
                                        BevyCjkAmbiguousWidth  cjk_ambiguous_width)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_enum (self->settings,
                       BEVY_PROFILE_KEY_CJK_AMBIGUOUS_WIDTH,
                       cjk_ambiguous_width);
}

gboolean
bevy_profile_get_bold_is_bright (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_BOLD_IS_BRIGHT);
}

void
bevy_profile_set_bold_is_bright (BevyProfile *self,
                                   gboolean       bold_is_bright)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_BOLD_IS_BRIGHT,
                          bold_is_bright);
}

double
bevy_profile_get_cell_height_scale (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_double (self->settings, BEVY_PROFILE_KEY_CELL_HEIGHT_SCALE);
}

void
bevy_profile_set_cell_height_scale (BevyProfile *self,
                                      double         cell_height_scale)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_double (self->settings,
                         BEVY_PROFILE_KEY_CELL_HEIGHT_SCALE,
                         cell_height_scale);
}

double
bevy_profile_get_cell_width_scale (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_double (self->settings, BEVY_PROFILE_KEY_CELL_WIDTH_SCALE);
}

void
bevy_profile_set_cell_width_scale (BevyProfile *self,
                                     double         cell_width_scale)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_double (self->settings,
                         BEVY_PROFILE_KEY_CELL_WIDTH_SCALE,
                         cell_width_scale);
}

int
bevy_profile_get_margin_x (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_int (self->settings, BEVY_PROFILE_KEY_MARGIN_X);
}

void
bevy_profile_set_margin_x (BevyProfile *self,
                            int         margin_x)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_int (self->settings, BEVY_PROFILE_KEY_MARGIN_X, margin_x);
}

int
bevy_profile_get_margin_y (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), 0);

  return g_settings_get_int (self->settings, BEVY_PROFILE_KEY_MARGIN_Y);
}

void
bevy_profile_set_margin_y (BevyProfile *self,
                            int         margin_y)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_int (self->settings, BEVY_PROFILE_KEY_MARGIN_Y, margin_y);
}

gboolean
bevy_profile_get_login_shell (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_LOGIN_SHELL);
}

void
bevy_profile_set_login_shell (BevyProfile *self,
                                gboolean       login_shell)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_LOGIN_SHELL,
                          login_shell);
}

gboolean
bevy_profile_get_use_custom_command (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_PROFILE_KEY_USE_CUSTOM_COMMAND);
}

void
bevy_profile_set_use_custom_command (BevyProfile *self,
                                       gboolean       use_custom_command)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_USE_CUSTOM_COMMAND,
                          use_custom_command);
}

gboolean
bevy_profile_get_use_proxy (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_PROFILE_KEY_USE_PROXY);
}

void
bevy_profile_set_use_proxy (BevyProfile *self,
                              gboolean       use_proxy)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  g_settings_set_boolean (self->settings,
                          BEVY_PROFILE_KEY_USE_PROXY,
                          use_proxy);
}

char *
bevy_profile_dup_custom_command (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return g_settings_get_string (self->settings, BEVY_PROFILE_KEY_CUSTOM_COMMAND);
}

void
bevy_profile_set_custom_command (BevyProfile *self,
                                   const char    *custom_command)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  if (custom_command == NULL)
    custom_command = "";

  g_settings_set_string (self->settings, BEVY_PROFILE_KEY_CUSTOM_COMMAND, custom_command);
}

/**
 * bevy_profile_list_custom_links:
 * @self: a [class@Bevy.Profile]
 *
 * Returns: (transfer full):
 */
GListModel *
bevy_profile_list_custom_links (BevyProfile *self)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), NULL);

  return g_object_ref (G_LIST_MODEL (self->custom_links));
}

void
bevy_profile_add_custom_link (BevyProfile    *self,
                                BevyCustomLink *custom_link)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (BEVY_IS_CUSTOM_LINK (custom_link));

  g_list_store_append (self->custom_links, custom_link);
}

void
bevy_profile_undo_remove_custom_link (BevyProfile    *self,
                                        BevyCustomLink *custom_link,
                                        guint             index)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));
  g_return_if_fail (BEVY_IS_CUSTOM_LINK (custom_link));

  if (index <= g_list_model_get_n_items (G_LIST_MODEL (self->custom_links)))
    g_list_store_insert (self->custom_links, index, custom_link);
  else
    g_list_store_append (self->custom_links, custom_link);
}

gboolean
bevy_profile_remove_custom_link (BevyProfile    *self,
                                   BevyCustomLink *custom_link,
                                   guint            *index)
{
  g_return_val_if_fail (BEVY_IS_PROFILE (self), FALSE);

  if (!g_list_store_find (self->custom_links, custom_link, index))
    {
      g_warning ("Custom link %p not found in profile %s", custom_link, self->uuid);
      return FALSE;
    }

  g_list_store_remove (self->custom_links, *index);

  return TRUE;
}

void
bevy_profile_save_custom_link_changes (BevyProfile *self)
{
  g_return_if_fail (BEVY_IS_PROFILE (self));

  write_custom_links_to_settings (self);
}
