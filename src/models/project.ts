// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupResource, GroupClass } from "./group";

export interface ProjectResource extends GroupResource {
    groupClass: GroupClass.PROJECT;
}
