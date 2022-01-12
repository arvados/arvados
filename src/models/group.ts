// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    ResourceKind,
    ResourceWithProperties,
    RESOURCE_UUID_REGEX,
    ResourceObjectType,
    TrashableResource
} from "./resource";

export interface GroupResource extends TrashableResource, ResourceWithProperties {
    kind: ResourceKind.GROUP;
    name: string;
    groupClass: GroupClass | null;
    description: string;
    writableBy: string[];
    ensure_unique_name: boolean;
}

export enum GroupClass {
    PROJECT = 'project',
    FILTER  = 'filter',
    ROLE  = 'role',
}

export const BUILTIN_GROUP_IDS = [
    'fffffffffffffff',
    'anonymouspublic',
    '000000000000000',
]

export const isBuiltinGroup = (uuid: string) => {
    const match = RESOURCE_UUID_REGEX.exec(uuid);
    const parts = match ? match[0].split('-') : [];
    return parts.length === 3 && parts[1] === ResourceObjectType.GROUP && BUILTIN_GROUP_IDS.includes(parts[2]);
};
