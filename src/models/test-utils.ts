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
    href: "",
    isTrashed: false,
    kind: ResourceKind.GROUP,
    modifiedAt: "",
    modifiedByClientUuid: "",
    modifiedByUserUuid: "",
    name: "",
    ownerUuid: "",
    properties: "",
    trashAt: "",
    uuid: "",
    writableBy: [],
    ensure_unique_name: true,
    ...data
});

export const mockProjectResource = (data: Partial<ProjectResource> = {}): ProjectResource =>
    mockGroupResource({ ...data, groupClass: GroupClass.PROJECT }) as ProjectResource;

export const mockCommonResource = (data: Partial<Resource>): Resource => ({
    createdAt: "",
    etag: "",
    href: "",
    kind: ResourceKind.NONE,
    modifiedAt: "",
    modifiedByClientUuid: "",
    modifiedByUserUuid: "",
    ownerUuid: "",
    uuid: "",
    ...data
});
