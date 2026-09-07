/*
 * bevy-agent-impl.h
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

#include "bevy-agent-ipc.h"
#include "bevy-container-provider.h"

G_BEGIN_DECLS

#define BEVY_TYPE_AGENT_IMPL (bevy_agent_impl_get_type())

G_DECLARE_FINAL_TYPE (BevyAgentImpl, bevy_agent_impl, BEVY, AGENT_IMPL, BevyIpcAgentSkeleton)

BevyAgentImpl *bevy_agent_impl_get_default   (void);
BevyAgentImpl *bevy_agent_impl_new           (GError                  **error);
void             bevy_agent_impl_add_container (BevyAgentImpl          *self,
                                                  BevyIpcContainer       *container);
void             bevy_agent_impl_add_provider  (BevyAgentImpl          *self,
                                                  BevyContainerProvider  *provider);


G_END_DECLS
