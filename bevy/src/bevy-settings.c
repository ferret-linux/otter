/*
 * bevy-settings.c
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

#include <gio/gio.h>

#include "bevy-application.h"
#include "bevy-enums.h"
#include "bevy-settings.h"
#include "bevy-util.h"

struct _BevySettings
{
  GObject    parent_instance;
  GSettings *settings;
};

enum {
  PROP_0,
  PROP_AUDIBLE_BELL,
  PROP_CURSOR_BLINK_MODE,
  PROP_CURSOR_SHAPE,
  PROP_DEFAULT_PROFILE_UUID,
  PROP_DISABLE_PADDING,
  PROP_ENABLE_A11Y,
  PROP_ENABLE_ZOOM_SCROLL_CTRL,
  PROP_IGNORE_OSC_TITLE,
  PROP_FONT_DESC,
  PROP_FONT_NAME,
  PROP_INTERFACE_STYLE,
  PROP_NEW_TAB_POSITION,
  PROP_PROFILE_UUIDS,
  PROP_RESTORE_SESSION,
  PROP_RESTORE_WINDOW_SIZE,
  PROP_DEFAULT_COLUMNS,
  PROP_DEFAULT_ROWS,
  PROP_SCROLLBAR_POLICY,
  PROP_TAB_MIDDLE_CLICK,
  PROP_TEXT_BLINK_MODE,
  PROP_TOAST_ON_COPY_CLIPBOARD,
  PROP_USE_SYSTEM_FONT,
  PROP_VISUAL_BELL,
  PROP_VISUAL_PROCESS_LEADER,
  PROP_WORD_CHAR_EXCEPTIONS,
  PROP_INHIBIT_LOGOUT,
  N_PROPS
};

G_DEFINE_FINAL_TYPE (BevySettings, bevy_settings, G_TYPE_OBJECT)

static GParamSpec *properties [N_PROPS];

static void
bevy_settings_changed_cb (BevySettings *self,
                            const char     *key,
                            GSettings      *settings)
{
  g_assert (BEVY_IS_SETTINGS (self));
  g_assert (key != NULL);
  g_assert (G_IS_SETTINGS (settings));

  if (g_str_equal (key, BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DEFAULT_PROFILE_UUID]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_DISABLE_PADDING))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DISABLE_PADDING]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_PROFILE_UUIDS))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_PROFILE_UUIDS]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_NEW_TAB_POSITION))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_NEW_TAB_POSITION]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_AUDIBLE_BELL))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_AUDIBLE_BELL]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_VISUAL_BELL))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_VISUAL_BELL]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_VISUAL_PROCESS_LEADER))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_VISUAL_PROCESS_LEADER]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_CURSOR_SHAPE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CURSOR_SHAPE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_CURSOR_BLINK_MODE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_CURSOR_BLINK_MODE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_SCROLLBAR_POLICY))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_SCROLLBAR_POLICY]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_TAB_MIDDLE_CLICK))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TAB_MIDDLE_CLICK]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_TEXT_BLINK_MODE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TEXT_BLINK_MODE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_INTERFACE_STYLE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_INTERFACE_STYLE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_RESTORE_SESSION))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_RESTORE_SESSION]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_RESTORE_WINDOW_SIZE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_RESTORE_WINDOW_SIZE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_DEFAULT_COLUMNS))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DEFAULT_COLUMNS]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_DEFAULT_ROWS))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_DEFAULT_ROWS]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_TOAST_ON_COPY_CLIPBOARD))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_TOAST_ON_COPY_CLIPBOARD]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_ENABLE_A11Y))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ENABLE_A11Y]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_ENABLE_ZOOM_SCROLL_CTRL))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_ENABLE_ZOOM_SCROLL_CTRL]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_IGNORE_OSC_TITLE))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_IGNORE_OSC_TITLE]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_INHIBIT_LOGOUT))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_INHIBIT_LOGOUT]);
  else if (g_str_equal (key, BEVY_SETTING_KEY_FONT_NAME))
    {
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_FONT_NAME]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_FONT_DESC]);
    }
  else if (g_str_equal (key, BEVY_SETTING_KEY_USE_SYSTEM_FONT))
    {
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_USE_SYSTEM_FONT]);
      g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_FONT_DESC]);
    }
  else if (g_str_equal (key, BEVY_SETTING_KEY_WORD_CHAR_EXCEPTIONS))
    g_object_notify_by_pspec (G_OBJECT (self), properties[PROP_WORD_CHAR_EXCEPTIONS]);
}

static void
bevy_settings_dispose (GObject *object)
{
  BevySettings *self = (BevySettings *)object;

  g_clear_object (&self->settings);

  G_OBJECT_CLASS (bevy_settings_parent_class)->dispose (object);
}

static void
bevy_settings_get_property (GObject    *object,
                              guint       prop_id,
                              GValue     *value,
                              GParamSpec *pspec)
{
  BevySettings *self = BEVY_SETTINGS (object);

  switch (prop_id)
    {
    case PROP_AUDIBLE_BELL:
      g_value_set_boolean (value, bevy_settings_get_audible_bell (self));
      break;

    case PROP_CURSOR_BLINK_MODE:
      g_value_set_enum (value, bevy_settings_get_cursor_blink_mode (self));
      break;

    case PROP_CURSOR_SHAPE:
      g_value_set_enum (value, bevy_settings_get_cursor_shape (self));
      break;

    case PROP_DEFAULT_PROFILE_UUID:
      g_value_take_string (value, bevy_settings_dup_default_profile_uuid (self));
      break;

    case PROP_DISABLE_PADDING:
      g_value_set_boolean (value, bevy_settings_get_disable_padding (self));
      break;

    case PROP_ENABLE_A11Y:
      g_value_set_boolean (value, bevy_settings_get_enable_a11y (self));
      break;

    case PROP_IGNORE_OSC_TITLE:
      g_value_set_boolean (value, bevy_settings_get_ignore_osc_title (self));
      break;

    case PROP_INHIBIT_LOGOUT:
      g_value_set_boolean (value, bevy_settings_get_inhibit_logout (self));
      break;

    case PROP_FONT_DESC:
      g_value_take_boxed (value, bevy_settings_dup_font_desc (self));
      break;

    case PROP_INTERFACE_STYLE:
      g_value_set_enum (value, bevy_settings_get_interface_style (self));
      break;

    case PROP_FONT_NAME:
      g_value_take_string (value, bevy_settings_dup_font_name (self));
      break;

    case PROP_NEW_TAB_POSITION:
      g_value_set_enum (value, bevy_settings_get_new_tab_position (self));
      break;

    case PROP_PROFILE_UUIDS:
      g_value_take_boxed (value, bevy_settings_dup_profile_uuids (self));
      break;

    case PROP_RESTORE_SESSION:
      g_value_set_boolean (value, bevy_settings_get_restore_session (self));
      break;

    case PROP_RESTORE_WINDOW_SIZE:
      g_value_set_boolean (value, bevy_settings_get_restore_window_size (self));
      break;

    case PROP_DEFAULT_COLUMNS:
      g_value_set_uint (value, bevy_settings_get_default_columns (self));
      break;

    case PROP_DEFAULT_ROWS:
      g_value_set_uint (value, bevy_settings_get_default_rows (self));
      break;

    case PROP_SCROLLBAR_POLICY:
      g_value_set_enum (value, bevy_settings_get_scrollbar_policy (self));
      break;

    case PROP_TAB_MIDDLE_CLICK:
      g_value_set_enum (value, bevy_settings_get_tab_middle_click (self));
      break;

    case PROP_TEXT_BLINK_MODE:
      g_value_set_enum (value, bevy_settings_get_text_blink_mode (self));
      break;

    case PROP_TOAST_ON_COPY_CLIPBOARD:
      g_value_set_boolean (value, bevy_settings_get_toast_on_copy_clipboard (self));
      break;

    case PROP_USE_SYSTEM_FONT:
      g_value_set_boolean (value, bevy_settings_get_use_system_font (self));
      break;

    case PROP_VISUAL_BELL:
      g_value_set_boolean (value, bevy_settings_get_visual_bell (self));
      break;

    case PROP_VISUAL_PROCESS_LEADER:
      g_value_set_boolean (value, bevy_settings_get_visual_process_leader (self));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_settings_set_property (GObject      *object,
                              guint         prop_id,
                              const GValue *value,
                              GParamSpec   *pspec)
{
  BevySettings *self = BEVY_SETTINGS (object);

  switch (prop_id)
    {
    case PROP_AUDIBLE_BELL:
      bevy_settings_set_audible_bell (self, g_value_get_boolean (value));
      break;

    case PROP_CURSOR_BLINK_MODE:
      bevy_settings_set_cursor_blink_mode (self, g_value_get_enum (value));
      break;

    case PROP_CURSOR_SHAPE:
      bevy_settings_set_cursor_shape (self, g_value_get_enum (value));
      break;

    case PROP_FONT_DESC:
      bevy_settings_set_font_desc (self, g_value_get_boxed (value));
      break;

    case PROP_ENABLE_A11Y:
      bevy_settings_set_enable_a11y (self, g_value_get_boolean (value));
      break;

    case PROP_ENABLE_ZOOM_SCROLL_CTRL:
      bevy_settings_set_enable_zoom_scroll_ctrl (self, g_value_get_boolean (value));
      break;

    case PROP_IGNORE_OSC_TITLE:
      bevy_settings_set_ignore_osc_title (self, g_value_get_boolean (value));
      break;

    case PROP_INHIBIT_LOGOUT:
      bevy_settings_set_inhibit_logout (self, g_value_get_boolean (value));
      break;

    case PROP_FONT_NAME:
      bevy_settings_set_font_name (self, g_value_get_string (value));
      break;

    case PROP_INTERFACE_STYLE:
      bevy_settings_set_interface_style (self, g_value_get_enum (value));
      break;

    case PROP_NEW_TAB_POSITION:
      bevy_settings_set_new_tab_position (self, g_value_get_enum (value));
      break;

    case PROP_DEFAULT_PROFILE_UUID:
      bevy_settings_set_default_profile_uuid (self, g_value_get_string (value));
      break;

    case PROP_DISABLE_PADDING:
      bevy_settings_set_disable_padding (self, g_value_get_boolean (value));
      break;

    case PROP_RESTORE_SESSION:
      bevy_settings_set_restore_session (self, g_value_get_boolean (value));
      break;

    case PROP_RESTORE_WINDOW_SIZE:
      bevy_settings_set_restore_window_size (self, g_value_get_boolean (value));
      break;

    case PROP_DEFAULT_COLUMNS:
      bevy_settings_set_default_columns (self, g_value_get_uint (value));
      break;

    case PROP_DEFAULT_ROWS:
      bevy_settings_set_default_rows (self, g_value_get_uint (value));
      break;

    case PROP_SCROLLBAR_POLICY:
      bevy_settings_set_scrollbar_policy (self, g_value_get_enum (value));
      break;

    case PROP_TAB_MIDDLE_CLICK:
      bevy_settings_set_tab_middle_click (self, g_value_get_enum (value));
      break;

    case PROP_TEXT_BLINK_MODE:
      bevy_settings_set_text_blink_mode (self, g_value_get_enum (value));
      break;

    case PROP_TOAST_ON_COPY_CLIPBOARD:
      bevy_settings_set_toast_on_copy_clipboard (self, g_value_get_boolean (value));
      break;

    case PROP_USE_SYSTEM_FONT:
      bevy_settings_set_use_system_font (self, g_value_get_boolean (value));
      break;

    case PROP_VISUAL_BELL:
      bevy_settings_set_visual_bell (self, g_value_get_boolean (value));
      break;

    case PROP_VISUAL_PROCESS_LEADER:
      bevy_settings_set_visual_process_leader (self, g_value_get_boolean (value));
      break;

    default:
      G_OBJECT_WARN_INVALID_PROPERTY_ID (object, prop_id, pspec);
    }
}

static void
bevy_settings_class_init (BevySettingsClass *klass)
{
  GObjectClass *object_class = G_OBJECT_CLASS (klass);

  object_class->dispose = bevy_settings_dispose;
  object_class->get_property = bevy_settings_get_property;
  object_class->set_property = bevy_settings_set_property;

  properties[PROP_AUDIBLE_BELL] =
    g_param_spec_boolean (BEVY_SETTING_KEY_AUDIBLE_BELL, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_CURSOR_BLINK_MODE] =
    g_param_spec_enum (BEVY_SETTING_KEY_CURSOR_BLINK_MODE, NULL, NULL,
                       VTE_TYPE_CURSOR_BLINK_MODE,
                       VTE_CURSOR_BLINK_SYSTEM,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_CURSOR_SHAPE] =
    g_param_spec_enum (BEVY_SETTING_KEY_CURSOR_SHAPE, NULL, NULL,
                       VTE_TYPE_CURSOR_SHAPE,
                       VTE_CURSOR_SHAPE_BLOCK,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_ENABLE_A11Y] =
    g_param_spec_boolean (BEVY_SETTING_KEY_ENABLE_A11Y, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_ENABLE_ZOOM_SCROLL_CTRL] =
    g_param_spec_boolean (BEVY_SETTING_KEY_ENABLE_ZOOM_SCROLL_CTRL, NULL, NULL,
                          FALSE,
                          (G_PARAM_READABLE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_IGNORE_OSC_TITLE] =
    g_param_spec_boolean (BEVY_SETTING_KEY_IGNORE_OSC_TITLE, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_INHIBIT_LOGOUT] =
    g_param_spec_boolean (BEVY_SETTING_KEY_INHIBIT_LOGOUT, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_FONT_DESC] =
    g_param_spec_boxed ("font-desc", NULL, NULL,
                        PANGO_TYPE_FONT_DESCRIPTION,
                        (G_PARAM_READWRITE |
                         G_PARAM_EXPLICIT_NOTIFY |
                         G_PARAM_STATIC_STRINGS));

  properties[PROP_FONT_NAME] =
    g_param_spec_string (BEVY_SETTING_KEY_FONT_NAME, NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_INTERFACE_STYLE] =
    g_param_spec_enum (BEVY_SETTING_KEY_INTERFACE_STYLE, NULL, NULL,
                       ADW_TYPE_COLOR_SCHEME,
                       ADW_COLOR_SCHEME_DEFAULT,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_NEW_TAB_POSITION] =
    g_param_spec_enum (BEVY_SETTING_KEY_NEW_TAB_POSITION, NULL, NULL,
                       BEVY_TYPE_NEW_TAB_POSITION,
                       BEVY_NEW_TAB_POSITION_LAST,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_DEFAULT_PROFILE_UUID] =
    g_param_spec_string (BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID, NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  properties[PROP_DISABLE_PADDING] =
    g_param_spec_boolean (BEVY_SETTING_KEY_DISABLE_PADDING, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_PROFILE_UUIDS] =
    g_param_spec_boxed (BEVY_SETTING_KEY_PROFILE_UUIDS, NULL, NULL,
                        G_TYPE_STRV,
                        (G_PARAM_READABLE |
                         G_PARAM_STATIC_STRINGS));

  properties[PROP_RESTORE_SESSION] =
    g_param_spec_boolean (BEVY_SETTING_KEY_RESTORE_SESSION, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_RESTORE_WINDOW_SIZE] =
    g_param_spec_boolean (BEVY_SETTING_KEY_RESTORE_WINDOW_SIZE, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_DEFAULT_COLUMNS] =
    g_param_spec_uint (BEVY_SETTING_KEY_DEFAULT_COLUMNS, NULL, NULL,
                       1, 65535, 80,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_DEFAULT_ROWS] =
    g_param_spec_uint (BEVY_SETTING_KEY_DEFAULT_ROWS, NULL, NULL,
                       1, 65535, 24,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_SCROLLBAR_POLICY] =
    g_param_spec_enum (BEVY_SETTING_KEY_SCROLLBAR_POLICY, NULL, NULL,
                       BEVY_TYPE_SCROLLBAR_POLICY,
                       BEVY_SCROLLBAR_POLICY_SYSTEM,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_TAB_MIDDLE_CLICK] =
    g_param_spec_enum (BEVY_SETTING_KEY_TAB_MIDDLE_CLICK, NULL, NULL,
                       BEVY_TYPE_TAB_MIDDLE_CLICK_BEHAVIOR,
                       BEVY_TAB_MIDDLE_CLICK_CLOSE,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_TEXT_BLINK_MODE] =
    g_param_spec_enum (BEVY_SETTING_KEY_TEXT_BLINK_MODE, NULL, NULL,
                       VTE_TYPE_TEXT_BLINK_MODE,
                       VTE_TEXT_BLINK_ALWAYS,
                       (G_PARAM_READWRITE |
                        G_PARAM_EXPLICIT_NOTIFY |
                        G_PARAM_STATIC_STRINGS));

  properties[PROP_TOAST_ON_COPY_CLIPBOARD] =
    g_param_spec_boolean (BEVY_SETTING_KEY_TOAST_ON_COPY_CLIPBOARD, NULL, NULL,
                          TRUE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_USE_SYSTEM_FONT] =
    g_param_spec_boolean (BEVY_SETTING_KEY_USE_SYSTEM_FONT, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_VISUAL_BELL] =
    g_param_spec_boolean (BEVY_SETTING_KEY_VISUAL_BELL, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_VISUAL_PROCESS_LEADER] =
    g_param_spec_boolean (BEVY_SETTING_KEY_VISUAL_PROCESS_LEADER, NULL, NULL,
                          FALSE,
                          (G_PARAM_READWRITE |
                           G_PARAM_EXPLICIT_NOTIFY |
                           G_PARAM_STATIC_STRINGS));

  properties[PROP_WORD_CHAR_EXCEPTIONS] =
    g_param_spec_string (BEVY_SETTING_KEY_WORD_CHAR_EXCEPTIONS, NULL, NULL,
                         NULL,
                         (G_PARAM_READWRITE |
                          G_PARAM_EXPLICIT_NOTIFY |
                          G_PARAM_STATIC_STRINGS));

  g_object_class_install_properties (object_class, N_PROPS, properties);
}

static void
bevy_settings_init (BevySettings *self)
{
  self->settings = g_settings_new (APP_SCHEMA_ID);

  g_signal_connect_object (self->settings,
                           "changed",
                           G_CALLBACK (bevy_settings_changed_cb),
                           self,
                           G_CONNECT_SWAPPED);
}

BevySettings *
bevy_settings_new (void)
{
  return g_object_new (BEVY_TYPE_SETTINGS, NULL);
}

void
bevy_settings_set_default_profile_uuid (BevySettings *self,
                                          const char     *default_profile_uuid)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));
  g_return_if_fail (default_profile_uuid != NULL);

  g_settings_set_string (self->settings,
                         BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID,
                         default_profile_uuid);
}

char *
bevy_settings_dup_default_profile_uuid (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  return g_settings_get_string (self->settings,
                                BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID);
}

char **
bevy_settings_dup_profile_uuids (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  return g_settings_get_strv (self->settings,
                              BEVY_SETTING_KEY_PROFILE_UUIDS);
}

void
bevy_settings_add_profile_uuid (BevySettings *self,
                                  const char     *uuid)
{
  g_auto(GStrv) profiles = NULL;
  gsize len;

  g_return_if_fail (BEVY_IS_SETTINGS (self));
  g_return_if_fail (uuid != NULL);

  profiles = g_settings_get_strv (self->settings,
                                  BEVY_SETTING_KEY_PROFILE_UUIDS);

  if (g_strv_contains ((const char * const *)profiles, uuid))
    return;

  len = g_strv_length (profiles);
  profiles = g_realloc_n (profiles, len + 2, sizeof (char *));
  profiles[len] = g_strdup (uuid);
  profiles[len+1] = NULL;

  g_settings_set_strv (self->settings,
                       BEVY_SETTING_KEY_PROFILE_UUIDS,
                       (const char * const *)profiles);
}

void
bevy_settings_remove_profile_uuid (BevySettings *self,
                                     const char     *uuid)
{
  g_auto(GStrv) profiles = NULL;
  g_autoptr(GStrvBuilder) builder = NULL;
  g_autofree char *default_profile_uuid = NULL;

  g_return_if_fail (BEVY_IS_SETTINGS (self));
  g_return_if_fail (uuid != NULL);

  default_profile_uuid = g_settings_get_string (self->settings,
                                                BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID);
  profiles = g_settings_get_strv (self->settings,
                                  BEVY_SETTING_KEY_PROFILE_UUIDS);

  builder = g_strv_builder_new ();
  for (guint i = 0; profiles[i]; i++)
    {
      if (!g_str_equal (profiles[i], uuid))
        g_strv_builder_add (builder, profiles[i]);
    }

  g_clear_pointer (&profiles, g_strfreev);
  profiles = g_strv_builder_end (builder);

  /* Make sure we have at least one profile */
  if (profiles[0] == NULL)
    {
      profiles = g_realloc_n (profiles, 2, sizeof (char *));
      profiles[0] = g_dbus_generate_guid ();
      profiles[1] = NULL;
    }

  g_settings_set_strv (self->settings,
                       BEVY_SETTING_KEY_PROFILE_UUIDS,
                       (const char * const *)profiles);

  if (g_str_equal (uuid, default_profile_uuid))
    g_settings_set_string (self->settings,
                           BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID,
                           profiles[0]);
}

BevyNewTabPosition
bevy_settings_get_new_tab_position (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_NEW_TAB_POSITION);
}

void
bevy_settings_set_new_tab_position (BevySettings       *self,
                                      BevyNewTabPosition  new_tab_position)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_NEW_TAB_POSITION,
                       new_tab_position);
}

GSettings *
bevy_settings_get_settings (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  return self->settings;
}

gboolean
bevy_settings_get_enable_a11y (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_ENABLE_A11Y);
}

void
bevy_settings_set_enable_a11y (BevySettings *self,
                                 gboolean        enable_a11y)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_ENABLE_A11Y,
                          enable_a11y);
}

gboolean
bevy_settings_get_enable_zoom_scroll_ctrl (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_SETTING_KEY_ENABLE_ZOOM_SCROLL_CTRL);
}

void
bevy_settings_set_enable_zoom_scroll_ctrl (BevySettings *self,
                                             gboolean        enable_zoom_scroll_ctrl)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_ENABLE_ZOOM_SCROLL_CTRL,
                          enable_zoom_scroll_ctrl);
}

gboolean
bevy_settings_get_audible_bell (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_AUDIBLE_BELL);
}

void
bevy_settings_set_audible_bell (BevySettings *self,
                                  gboolean        audible_bell)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_AUDIBLE_BELL,
                          audible_bell);
}

gboolean
bevy_settings_get_visual_bell (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_VISUAL_BELL);
}

void
bevy_settings_set_visual_bell (BevySettings *self,
                                 gboolean        visual_bell)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_VISUAL_BELL,
                          visual_bell);
}

gboolean
bevy_settings_get_visual_process_leader (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_VISUAL_PROCESS_LEADER);
}

void
bevy_settings_set_visual_process_leader (BevySettings *self,
                                           gboolean        visual_process_leader)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_VISUAL_PROCESS_LEADER,
                          visual_process_leader);
}

VteCursorBlinkMode
bevy_settings_get_cursor_blink_mode (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_CURSOR_BLINK_MODE);
}

void
bevy_settings_set_cursor_blink_mode (BevySettings     *self,
                                       VteCursorBlinkMode  cursor_blink_mode)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_CURSOR_BLINK_MODE,
                       cursor_blink_mode);
}

VteCursorShape
bevy_settings_get_cursor_shape (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_CURSOR_SHAPE);
}

void
bevy_settings_set_cursor_shape (BevySettings *self,
                                  VteCursorShape  cursor_shape)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_CURSOR_SHAPE,
                       cursor_shape);
}

char *
bevy_settings_dup_font_name (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  return g_settings_get_string (self->settings, BEVY_SETTING_KEY_FONT_NAME);
}

void
bevy_settings_set_font_name (BevySettings *self,
                               const char     *font_name)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  if (font_name == NULL)
    font_name = "";

  g_settings_set_string (self->settings, BEVY_SETTING_KEY_FONT_NAME, font_name);
}

gboolean
bevy_settings_get_use_system_font (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_USE_SYSTEM_FONT);
}

void
bevy_settings_set_use_system_font (BevySettings *self,
                                     gboolean        use_system_font)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings, BEVY_SETTING_KEY_USE_SYSTEM_FONT, use_system_font);
}

PangoFontDescription *
bevy_settings_dup_font_desc (BevySettings *self)
{
  BevyApplication *app;
  g_autofree char *font_name = NULL;
  const char *system_font_name;

  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  app = BEVY_APPLICATION_DEFAULT;
  system_font_name = bevy_application_get_system_font_name (app);

  if (bevy_settings_get_use_system_font (self) ||
      !(font_name = bevy_settings_dup_font_name (self)) ||
      bevy_str_empty0 (font_name))
    return pango_font_description_from_string (system_font_name);

  return pango_font_description_from_string (font_name);
}

void
bevy_settings_set_font_desc (BevySettings             *self,
                               const PangoFontDescription *font_desc)
{
  g_autofree char *font_name = NULL;

  g_return_if_fail (BEVY_IS_SETTINGS (self));

  if (font_desc != NULL)
    font_name = pango_font_description_to_string (font_desc);

  bevy_settings_set_font_name (self, font_name);
}

BevyScrollbarPolicy
bevy_settings_get_scrollbar_policy (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_SCROLLBAR_POLICY);
}

void
bevy_settings_set_scrollbar_policy (BevySettings        *self,
                                      BevyScrollbarPolicy  scrollbar_policy)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_SCROLLBAR_POLICY,
                       scrollbar_policy);
}

BevyTabMiddleClickBehavior
bevy_settings_get_tab_middle_click (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_TAB_MIDDLE_CLICK);
}

void
bevy_settings_set_tab_middle_click (BevySettings               *self,
                                      BevyTabMiddleClickBehavior  tab_middle_click)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_TAB_MIDDLE_CLICK,
                       tab_middle_click);
}

VteTextBlinkMode
bevy_settings_get_text_blink_mode (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_TEXT_BLINK_MODE);
}

void
bevy_settings_set_text_blink_mode (BevySettings   *self,
                                     VteTextBlinkMode  text_blink_mode)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_enum (self->settings,
                       BEVY_SETTING_KEY_TEXT_BLINK_MODE,
                       text_blink_mode);
}

gboolean
bevy_settings_get_restore_session (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_RESTORE_SESSION);
}

void
bevy_settings_set_restore_session (BevySettings *self,
                                     gboolean        restore_session)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_RESTORE_SESSION,
                          restore_session);
}

gboolean
bevy_settings_get_restore_window_size (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_RESTORE_WINDOW_SIZE);
}

void
bevy_settings_set_restore_window_size (BevySettings *self,
                                         gboolean        restore_window_size)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_RESTORE_WINDOW_SIZE,
                          restore_window_size);
}

void
bevy_settings_get_default_size (BevySettings *self,
                                  guint          *columns,
                                  guint          *rows)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  if (columns != NULL)
    *columns = bevy_settings_get_default_columns (self);

  if (rows != NULL)
    *rows = bevy_settings_get_default_rows (self);
}

guint
bevy_settings_get_default_columns (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_uint (self->settings, BEVY_SETTING_KEY_DEFAULT_COLUMNS);
}

guint
bevy_settings_get_default_rows (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_uint (self->settings, BEVY_SETTING_KEY_DEFAULT_ROWS);
}

void
bevy_settings_set_default_columns (BevySettings *self,
                                     guint          columns)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_uint (self->settings,
                      BEVY_SETTING_KEY_DEFAULT_COLUMNS,
                      columns);
}

void
bevy_settings_set_default_rows (BevySettings *self,
                                  guint          rows)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_uint (self->settings,
                      BEVY_SETTING_KEY_DEFAULT_ROWS,
                      rows);
}

void
bevy_settings_get_window_size (BevySettings *self,
                                 guint          *columns,
                                 guint          *rows)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));
  g_return_if_fail (columns != NULL);
  g_return_if_fail (rows != NULL);

  g_settings_get (self->settings, "window-size", "(uu)", columns, rows);
}

void
bevy_settings_set_window_size (BevySettings *self,
                                 guint           columns,
                                 guint           rows)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set (self->settings, "window-size", "(uu)", columns, rows);
}

AdwColorScheme
bevy_settings_get_interface_style (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), 0);

  return g_settings_get_enum (self->settings, BEVY_SETTING_KEY_INTERFACE_STYLE);
}

void
bevy_settings_set_interface_style (BevySettings *self,
                                     AdwColorScheme  color_scheme)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));
  g_return_if_fail (color_scheme == ADW_COLOR_SCHEME_DEFAULT ||
                    color_scheme == ADW_COLOR_SCHEME_FORCE_LIGHT ||
                    color_scheme == ADW_COLOR_SCHEME_FORCE_DARK);

  g_settings_set_enum (self->settings, BEVY_SETTING_KEY_INTERFACE_STYLE, color_scheme);
}

gboolean
bevy_settings_get_toast_on_copy_clipboard (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings, BEVY_SETTING_KEY_TOAST_ON_COPY_CLIPBOARD);
}

void
bevy_settings_set_toast_on_copy_clipboard (BevySettings *self,
                                             gboolean        toast_on_copy_clipboard)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_TOAST_ON_COPY_CLIPBOARD,
                          toast_on_copy_clipboard);
}

void
bevy_settings_set_disable_padding (BevySettings *self,
                                     gboolean        disable_padding)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_DISABLE_PADDING,
                          !!disable_padding);
}

gboolean
bevy_settings_get_disable_padding (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_SETTING_KEY_DISABLE_PADDING);
}

char *
bevy_settings_dup_word_char_exceptions (BevySettings *self)
{
  char *word_char_exceptions;

  g_return_val_if_fail (BEVY_IS_SETTINGS (self), NULL);

  g_settings_get (self->settings, BEVY_SETTING_KEY_WORD_CHAR_EXCEPTIONS, "ms", &word_char_exceptions);
  return word_char_exceptions;
}

void
bevy_settings_set_prompt_on_close (BevySettings *self,
                                     gboolean        prompt_on_close)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_PROMPT_ON_CLOSE,
                          !!prompt_on_close);
}

gboolean
bevy_settings_get_prompt_on_close (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_SETTING_KEY_PROMPT_ON_CLOSE);
}

void
bevy_settings_set_ignore_osc_title (BevySettings *self,
                                      gboolean        ignore_osc_title)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_IGNORE_OSC_TITLE,
                          !!ignore_osc_title);
}

gboolean
bevy_settings_get_ignore_osc_title (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_SETTING_KEY_IGNORE_OSC_TITLE);
}

void
bevy_settings_set_inhibit_logout (BevySettings *self,
                                    gboolean        inhibit_logout)
{
  g_return_if_fail (BEVY_IS_SETTINGS (self));

  g_settings_set_boolean (self->settings,
                          BEVY_SETTING_KEY_INHIBIT_LOGOUT,
                          !!inhibit_logout);
}

gboolean
bevy_settings_get_inhibit_logout (BevySettings *self)
{
  g_return_val_if_fail (BEVY_IS_SETTINGS (self), FALSE);

  return g_settings_get_boolean (self->settings,
                                 BEVY_SETTING_KEY_INHIBIT_LOGOUT);
}
