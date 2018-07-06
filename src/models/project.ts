// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource as R } from "./resource";
import { GroupResource } from "./group";

export interface Project extends R {
}

export interface ProjectResource extends GroupResource {
    groupClass: "project";
}
