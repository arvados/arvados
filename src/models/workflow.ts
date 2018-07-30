// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export interface WorkflowResource extends Resource {
    kind: ResourceKind.WORKFLOW;
    name: string;
    description: string;
    definition: string;
}