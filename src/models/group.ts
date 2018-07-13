// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export interface GroupResource extends Resource {
    kind: ResourceKind.Group;
    name: string;
    groupClass: GroupClass | null;
    description: string;
    properties: string;
    writeableBy: string[];
    trashAt: string;
    deleteAt: string;
    isTrashed: boolean;
}

export enum GroupClass {
    Project = "project"
}