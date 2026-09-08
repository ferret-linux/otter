/* test-palette-search.c
 *
 * Copyright 2026 Noel <noel@example.com>
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

#include <glib.h>

#include "bevy-palette-search.h"

/*
 * Load the real bundled palette names so the search logic is exercised
 * against actual production data rather than a hand-picked sample.
 */
static char **
load_palette_names (gsize *n_out)
{
  const char *srcdir;
  g_autofree char *palette_dir = NULL;
  GDir *dir;
  GPtrArray *names = g_ptr_array_new_with_free_func (g_free);
  const char *filename;
  GError *error = NULL;

  srcdir = g_getenv ("G_TEST_SRCDIR");
  g_assert_nonnull (srcdir);

  palette_dir = g_build_filename (srcdir, "..", "src", "palettes", NULL);
  dir = g_dir_open (palette_dir, 0, &error);
  g_assert_no_error (error);
  g_assert_nonnull (dir);

  while ((filename = g_dir_read_name (dir)))
    {
      g_autofree char *path = NULL;
      g_autofree char *contents = NULL;
      char **lines;
      gsize len;

      if (!g_str_has_suffix (filename, ".palette"))
        continue;

      path = g_build_filename (palette_dir, filename, NULL);
      if (!g_file_get_contents (path, &contents, &len, &error))
        {
          g_assert_no_error (error);
          continue;
        }

      lines = g_strsplit (contents, "\n", -1);
      for (gsize i = 0; lines[i] != NULL; i++)
        {
          if (g_str_has_prefix (lines[i], "Name="))
            {
              const char *name = lines[i] + strlen ("Name=");
              if (name[0] != '\0')
                g_ptr_array_add (names, g_strdup (name));
              break;
            }
        }
      g_strfreev (lines);
    }

  g_dir_close (dir);
  *n_out = names->len;
  return (char **)g_ptr_array_free (names, FALSE);
}

static void
test_fuzzy_matches (void)
{
  gsize n;
  char **names = load_palette_names (&n);
  const char *query;
  const char *want;
  const char *schema[] = {
    "nord",     "Nord",
    "aglow",    "Afterglow",
    "gruv",     "Gruvbox",
    "gruvbox",  "Gruvbox",
    "grubox",   "Gruvbox",
    "tokyo",    "Tokyo Night",
    "tokyo n",  "Tokyo Night",
    "everf",    "Everforest",
    "catp",     "Catppuccin Mocha",
    "solar",    "Solarized",
    "dracul",   "Dracula",
    NULL,       NULL,
  };

  g_assert_cmpuint (n, >, 100); /* ensure we really loaded the palette set */

  for (gsize i = 0; schema[i] != NULL; i += 2)
    {
      query = schema[i];
      want = schema[i + 1];

      for (gsize k = 0; k < n; k++)
        {
          if (g_strcmp0 (names[k], want) == 0)
            {
              int score = bevy_palette_search_score (query, names[k]);
              g_assert_cmpint (score, >, 0);
              break;
            }

          if (k == n - 1)
            g_error ("expected palette \"%s\" not found in data set", want);
        }
    }

  g_strfreev (names);
}

static void
test_fuzzy_ranking (void)
{
  gsize n;
  char **names = load_palette_names (&n);
  const char *queries[] = { "nord", "aglow", NULL };
  const char *expected_first[] = { "Nord", "Afterglow", NULL };

  for (gsize qi = 0; queries[qi] != NULL; qi++)
    {
      const char *query = queries[qi];
      const char *best_name = NULL;
      int best_score = 0;

      for (gsize k = 0; k < n; k++)
        {
          int score = bevy_palette_search_score (query, names[k]);
          if (score > best_score)
            {
              best_score = score;
              best_name = names[k];
            }
        }

      g_assert_nonnull (best_name);
      g_assert_cmpstr (best_name, ==, expected_first[qi]);
    }

  g_strfreev (names);
}

static void
test_no_false_positives (void)
{
  const char *names[] = { "Dracula", "Nord", "Gruvbox", "Solarized", "Tokyo Night" };
  const char *queries[] = { "zzzzzz", "xyzzy", "northwind", NULL };

  for (gsize qi = 0; queries[qi] != NULL; qi++)
    for (gsize ki = 0; ki < G_N_ELEMENTS (names); ki++)
      g_assert_cmpint (bevy_palette_search_score (queries[qi], names[ki]), ==, 0);
}

int
main (int argc,
      char *argv[])
{
  g_test_init (&argc, &argv, NULL);
  g_test_add_func ("/Bevy/PaletteSearch/FuzzyMatches", test_fuzzy_matches);
  g_test_add_func ("/Bevy/PaletteSearch/FuzzyRanking", test_fuzzy_ranking);
  g_test_add_func ("/Bevy/PaletteSearch/NoFalsePositives", test_no_false_positives);
  return g_test_run ();
}
