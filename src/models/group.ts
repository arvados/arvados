// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind, TrashableResource } from "./resource";

export interface GroupResource extends TrashableResource {
    kind: ResourceKind.GROUP;
    name: string;
    groupClass: GroupClass | null;
    description: string;
    properties: any;
    writeableBy: string[];
}

export enum GroupClass {
    PROJECT = "project"
}
