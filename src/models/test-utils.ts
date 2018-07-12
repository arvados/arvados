// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupResource } from "./group";
import { Resource, ResourceKind } from "./resource";

type ResourceUnion = GroupResource;

export const mockResource = (kind: ResourceKind, data: Partial<Exclude<ResourceUnion, "kind">>) => {
    switch (kind) {
        case ResourceKind.Group:
            return mockGroupResource({ ...data, kind });
        default:
            return mockCommonResource({ ...data, kind });
    }
};

export const mockGroupResource = (data: Partial<Exclude<GroupResource, "kind">>): GroupResource => ({
    createdAt: "",
    deleteAt: "",
    description: "",
    etag: "",
    groupClass: null,
    href: "",
    isTrashed: false,
    kind: ResourceKind.Group,
    modifiedAt: "",
    modifiedByClientUuid: "",
    modifiedByUserUuid: "",
    name: "",
    ownerUuid: "",
    properties: "",
    trashAt: "",
    uuid: "",
    writeableBy: []
});

const mockCommonResource = <T extends Resource>(data: Partial<T> & { kind: ResourceKind }): Resource => ({
    createdAt: "",
    etag: "",
    href: "",
    kind: data.kind,
    modifiedAt: "",
    modifiedByClientUuid: "",
    modifiedByUserUuid: "",
    ownerUuid: "",
    uuid: ""
});
