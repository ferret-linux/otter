/*
 * bevy-settings.h
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

#pragma once

#include <adwaita.h>
#include <vte/vte.h>

G_BEGIN_DECLS

#define BEVY_SETTING_KEY_AUDIBLE_BELL            "audible-bell"
#define BEVY_SETTING_KEY_CURSOR_BLINK_MODE       "cursor-blink-mode"
#define BEVY_SETTING_KEY_CURSOR_SHAPE            "cursor-shape"
#define BEVY_SETTING_KEY_DEFAULT_PROFILE_UUID    "default-profile-uuid"
#define BEVY_SETTING_KEY_ENABLE_A11Y             "enable-a11y"
#define BEVY_SETTING_KEY_ENABLE_ZOOM_SCROLL_CTRL "enable-zoom-scroll-ctrl"
#define BEVY_SETTING_KEY_FONT_NAME               "font-name"
#define BEVY_SETTING_KEY_INTERFACE_STYLE         "interface-style"
#define BEVY_SETTING_KEY_NEW_TAB_POSITION        "new-tab-position"
#define BEVY_SETTING_KEY_PROFILE_UUIDS           "profile-uuids"
#define BEVY_SETTING_KEY_PROMPT_ON_CLOSE         "prompt-on-close"
#define BEVY_SETTING_KEY_RESTORE_SESSION         "restore-session"
#define BEVY_SETTING_KEY_RESTORE_WINDOW_SIZE     "restore-window-size"
#define BEVY_SETTING_KEY_DEFAULT_COLUMNS         "default-columns"
#define BEVY_SETTING_KEY_DEFAULT_ROWS            "default-rows"
#define BEVY_SETTING_KEY_SCROLLBAR_POLICY        "scrollbar-policy"
#define BEVY_SETTING_KEY_TEXT_BLINK_MODE         "text-blink-mode"
#define BEVY_SETTING_KEY_TOAST_ON_COPY_CLIPBOARD "toast-on-copy-clipboard"
#define BEVY_SETTING_KEY_USE_SYSTEM_FONT         "use-system-font"
#define BEVY_SETTING_KEY_VISUAL_BELL             "visual-bell"
#define BEVY_SETTING_KEY_VISUAL_PROCESS_LEADER   "visual-process-leader"
#define BEVY_SETTING_KEY_WORD_CHAR_EXCEPTIONS    "word-char-exceptions"
#define BEVY_SETTING_KEY_TAB_MIDDLE_CLICK        "tab-middle-click"
#define BEVY_SETTING_KEY_IGNORE_OSC_TITLE        "ignore-osc-title"
#define BEVY_SETTING_KEY_INHIBIT_LOGOUT          "inhibit-logout"

typedef enum _BevyNewTabPosition
{
  BEVY_NEW_TAB_POSITION_LAST = 0,
  BEVY_NEW_TAB_POSITION_NEXT,
} BevyNewTabPosition;

typedef enum _BevyScrollbarPolicy
{
  BEVY_SCROLLBAR_POLICY_NEVER  = 0,
  BEVY_SCROLLBAR_POLICY_SYSTEM = 1,
  BEVY_SCROLLBAR_POLICY_ALWAYS = 2,
} BevyScrollbarPolicy;

/**
 * BevyTabMiddleClickBehavior:
 * %BEVY_TAB_MIDDLE_CLICK_CLOSE: close the tab on middle mouse click
 * %BEVY_TAB_MIDDLE_CLICK_PASTE: raise tab and paste clipboard contents on middle mouse click
 * %BEVY_TAB_MIDDLE_CLICK_NOTHING: raise tab only
 *
 * Enumeration describing the action to take on middle-mouse-click on tab widget
 */
typedef enum _BevyTabMiddleClickBehavior
{
  BEVY_TAB_MIDDLE_CLICK_CLOSE   = 0,
  BEVY_TAB_MIDDLE_CLICK_PASTE   = 1,
  BEVY_TAB_MIDDLE_CLICK_NOTHING = 2,
} BevyTabMiddleClickBehavior;

#define BEVY_TYPE_SETTINGS (bevy_settings_get_type())

G_DECLARE_FINAL_TYPE (BevySettings, bevy_settings, BEVY, SETTINGS, GObject)

BevySettings         *bevy_settings_new                         (void);
GSettings              *bevy_settings_get_settings                (BevySettings             *self);
char                   *bevy_settings_dup_default_profile_uuid    (BevySettings             *self);
void                    bevy_settings_set_default_profile_uuid    (BevySettings             *self,
                                                                     const char                 *uuid);
char                  **bevy_settings_dup_profile_uuids           (BevySettings             *self);
void                    bevy_settings_add_profile_uuid            (BevySettings             *self,
                                                                     const char                 *uuid);
void                    bevy_settings_remove_profile_uuid         (BevySettings             *self,
                                                                     const char                 *uuid);
BevyNewTabPosition    bevy_settings_get_new_tab_position        (BevySettings             *self);
void                    bevy_settings_set_new_tab_position        (BevySettings             *self,
                                                                     BevyNewTabPosition        new_tab_position);
gboolean                bevy_settings_get_enable_a11y             (BevySettings             *self);
void                    bevy_settings_set_enable_a11y             (BevySettings             *self,
                                                                     gboolean                    enable_a11y);
gboolean                bevy_settings_get_enable_zoom_scroll_ctrl (BevySettings             *self);
void                    bevy_settings_set_enable_zoom_scroll_ctrl (BevySettings             *self,
                                                                     gboolean                    enable_zoom_scroll_ctrl);
gboolean                bevy_settings_get_audible_bell            (BevySettings             *self);
void                    bevy_settings_set_audible_bell            (BevySettings             *self,
                                                                     gboolean                    audible_bell);
gboolean                bevy_settings_get_visual_bell             (BevySettings             *self);
void                    bevy_settings_set_visual_bell             (BevySettings             *self,
                                                                     gboolean                    visual_bell);
gboolean                bevy_settings_get_visual_process_leader   (BevySettings             *self);
void                    bevy_settings_set_visual_process_leader   (BevySettings             *self,
                                                                     gboolean                    visual_process_leader);
VteCursorBlinkMode      bevy_settings_get_cursor_blink_mode       (BevySettings             *self);
void                    bevy_settings_set_cursor_blink_mode       (BevySettings             *self,
                                                                     VteCursorBlinkMode          blink_mode);
VteCursorShape          bevy_settings_get_cursor_shape            (BevySettings             *self);
void                    bevy_settings_set_cursor_shape            (BevySettings             *self,
                                                                     VteCursorShape              cursor_shape);
PangoFontDescription   *bevy_settings_dup_font_desc               (BevySettings             *self);
void                    bevy_settings_set_font_desc               (BevySettings             *self,
                                                                     const PangoFontDescription *font_desc);
char                   *bevy_settings_dup_font_name               (BevySettings             *self);
void                    bevy_settings_set_font_name               (BevySettings             *self,
                                                                     const char                 *font_name);
gboolean                bevy_settings_get_use_system_font         (BevySettings             *self);
void                    bevy_settings_set_use_system_font         (BevySettings             *self,
                                                                     gboolean                    use_system_font);
gboolean                bevy_settings_get_restore_session         (BevySettings             *self);
void                    bevy_settings_set_restore_session         (BevySettings             *self,
                                                                     gboolean                    restore_session);
gboolean                bevy_settings_get_restore_window_size     (BevySettings             *self);
void                    bevy_settings_set_restore_window_size     (BevySettings             *self,
                                                                     gboolean                    restore_window_size);
BevyScrollbarPolicy   bevy_settings_get_scrollbar_policy        (BevySettings             *self);
void                    bevy_settings_set_scrollbar_policy        (BevySettings             *self,
                                                                     BevyScrollbarPolicy       scrollbar_policy);
BevyTabMiddleClickBehavior   bevy_settings_get_tab_middle_click        (BevySettings               *self);
void                           bevy_settings_set_tab_middle_click        (BevySettings               *self,
                                                                            BevyTabMiddleClickBehavior  tab_middle_click);
VteTextBlinkMode        bevy_settings_get_text_blink_mode         (BevySettings             *self);
void                    bevy_settings_set_text_blink_mode         (BevySettings             *self,
                                                                     VteTextBlinkMode            text_blink_mode);
void                    bevy_settings_get_window_size             (BevySettings             *self,
                                                                     guint                      *columns,
                                                                     guint                      *rows);
void                    bevy_settings_set_window_size             (BevySettings             *self,
                                                                     guint                       columns,
                                                                     guint                       rows);
void                    bevy_settings_get_default_size            (BevySettings             *self,
                                                                     guint                      *columns,
                                                                     guint                      *rows);
guint                   bevy_settings_get_default_columns         (BevySettings             *self);
void                    bevy_settings_set_default_columns         (BevySettings             *self,
                                                                     guint                      columns);
guint                   bevy_settings_get_default_rows            (BevySettings             *self);
void                    bevy_settings_set_default_rows            (BevySettings             *self,
                                                                     guint                      rows);
AdwColorScheme          bevy_settings_get_interface_style         (BevySettings             *self);
void                    bevy_settings_set_interface_style         (BevySettings             *self,
                                                                     AdwColorScheme              color_scheme);
gboolean                bevy_settings_get_toast_on_copy_clipboard (BevySettings             *self);
void                    bevy_settings_set_toast_on_copy_clipboard (BevySettings             *self,
                                                                     gboolean                    toast_on_copy_clipboard);
char                   *bevy_settings_dup_word_char_exceptions    (BevySettings             *self);
gboolean                bevy_settings_get_prompt_on_close         (BevySettings             *self);
void                    bevy_settings_set_prompt_on_close         (BevySettings             *self,
                                                                     gboolean                    prompt_on_close);
gboolean                bevy_settings_get_ignore_osc_title        (BevySettings             *self);
void                    bevy_settings_set_ignore_osc_title        (BevySettings             *self,
                                                                     gboolean                    ignore_osc_title);
gboolean                bevy_settings_get_inhibit_logout          (BevySettings             *self);
void                    bevy_settings_set_inhibit_logout          (BevySettings             *self,
                                                                     gboolean                    inhibit_logout);

G_END_DECLS
