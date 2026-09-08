/* bevy-palette-search.h
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

#pragma once

#include <glib.h>

G_BEGIN_DECLS

/* Score how well @query matches @name.
 *
 * Returns an integer where 0 means "not a match" and any positive value is a
 * match.  Higher values indicate a stronger match, suitable for relevance
 * ranking.  Matching is case-insensitive and tolerant of typos and stray
 * characters (fuzzy subsequence matching).
 */
int  bevy_palette_search_score (const char *query,
                                const char *name);

G_END_DECLS
