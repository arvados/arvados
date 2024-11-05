// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupClass, GroupResource } from "./group";
import { Resource, ResourceKind } from "./resource";
import { ProjectResource } from "./project";

export const mockGroupResource = (data: Partial<GroupResource> = {}): GroupResource => ({
    createdAt: "",
    deleteAt: "",
    description: "",
    etag: "",
    groupClass: null,
    isTrashed: false,
    kind: ResourceKind.GROUP,
    modifiedAt: "",
    modifiedByUserUuid: "",
    name: "",
    ownerUuid: "",
    properties: "",
    trashAt: "",
    uuid: "",
    ensure_unique_name: true,
    canWrite: false,
    canManage: false,
    ...data
});

export const mockProjectResource = (data: Partial<ProjectResource> = {}): ProjectResource =>
    mockGroupResource({ ...data, groupClass: GroupClass.PROJECT }) as ProjectResource;

export const mockCommonResource = (data: Partial<Resource>): Resource => ({
    createdAt: "",
    etag: "",
    kind: ResourceKind.NONE,
    modifiedAt: "",
    modifiedByUserUuid: "",
    ownerUuid: "",
    uuid: "",
    ...data
});
