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
    ensure_unique_name: boolean;
    canWrite: boolean;
    canManage: boolean;
}

export enum GroupClass {
    PROJECT = 'project',
    FILTER = 'filter',
    ROLE = 'role',
}

export enum BuiltinGroups {
    ALL = 'fffffffffffffff',
    ANON = 'anonymouspublic',
    SYSTEM = '000000000000000',
}

export const getBuiltinGroupUuid = (cluster: string, groupName: BuiltinGroups): string => {
    return cluster ? `${cluster}-${ResourceObjectType.GROUP}-${groupName}` : "";
};

export const isBuiltinGroup = (uuid: string) => {
    const match = RESOURCE_UUID_REGEX.exec(uuid);
    const parts = match ? match[0].split('-') : [];
    return parts.length === 3 && parts[1] === ResourceObjectType.GROUP && Object.values<string>(BuiltinGroups).includes(parts[2]);
};

export const selectedFieldsOfGroup = [
    "uuid",
    "name",
    "group_class",
    "description",
    "properties",
    "can_write",
    "can_manage",
    "trash_at",
    "delete_at",
    "is_trashed",
    "frozen_by_uuid"
];
