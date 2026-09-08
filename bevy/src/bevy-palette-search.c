/* bevy-palette-search.c
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

#include "config.h"

#include <string.h>

#include "bevy-palette-search.h"

/*
 * A single higher-is-better score returned by bevy_palette_search_score().
 *
 * - Exact substring/prefix matches are given a score around 100000, minus a
 *   penalty proportional to how far into the name the match begins.  Thematic
 *   palettes like "Nord" or "Tokyo Night" therefore rank at the top.
 * - Inexact "fuzzy" matches (characters in order but with gaps) get a score
 *   around 50000, anchored by how early the match starts and how few gaps it
 *   has.  This allows typos ("gruvbox" -> "Gruvbox") and abbreviations
 *   ("aglow" -> "Afterglow") to still surface, while keeping them ranked below
 *   exact substring matches.
 *
 * Any score below BEVY_PALETTE_SEARCH_MIN_SCORE is treated as "not a match",
 * which filters out the loose false positives that a pure subsequence match
 * produces (e.g. "nord" matching "Mono Red").
 */
#define BEVY_PALETTE_SEARCH_MIN_SCORE  52000

/*
 * Copy @src into @dst (at most @dst_size bytes) keeping only characters that
 * are useful for matching: everything is lowercased, whitespace collapses to a
 * single space, and separator characters (-, _, .) are stripped.
 */
static void
normalize (const char *src,
           char       *dst,
           gsize       dst_size)
{
  gsize in = 0;
  gsize out = 0;

  for (in = 0; src[in] != '\0' && out + 1 < dst_size; in++)
    {
      char ch = (char)g_ascii_tolower ((guchar)src[in]);

      if (ch == '-' || ch == '_' || ch == '.')
        continue;

      if (ch == ' ')
        {
          if (out > 0 && dst[out - 1] != ' ')
            dst[out++] = ' ';
          continue;
        }

      dst[out++] = ch;
    }

  dst[out] = '\0';
}

int
bevy_palette_search_score (const char *query,
                           const char *name)
{
  char q[256];
  char s[256];
  const char *sub;
  int qlen;
  int slen;
  int i;
  int j;
  int gaps = 0;
  int last = -1;
  int start;
  int score;

  g_return_val_if_fail (query != NULL, 0);
  g_return_val_if_fail (name != NULL, 0);

  normalize (query, q, sizeof q);
  normalize (name, s, sizeof s);

  qlen = (int)strlen (q);
  slen = (int)strlen (s);

  if (qlen == 0 || slen == 0 || qlen > slen)
    return 0;

  /* Exact substring match. */
  sub = strstr (s, q);
  if (sub != NULL)
    {
      int pos = (int)(sub - s);
      score = 100000 - pos;
      if (pos == 0)
        score += 5000;
      else if (s[pos - 1] == ' ')
        score += 2000;
      return score;
    }

  /* Fuzzy subsequence match: every query char appears in order, gaps allowed. */
  for (i = 0, j = 0; i < qlen && j < slen; j++)
    {
      if (q[i] == s[j])
        {
          if (last >= 0)
            gaps += (j - last - 1);
          last = j;
          i++;
        }
    }

  if (i < qlen)
    return 0;

  for (start = 0; start < slen && s[start] != q[0]; start++)
    ;

  score = 50000 - start * 100 - gaps * 200;
  if (start == 0 || s[start - 1] == ' ')
    score += 8000;

  if (score < BEVY_PALETTE_SEARCH_MIN_SCORE)
    return 0;

  return score;
}
