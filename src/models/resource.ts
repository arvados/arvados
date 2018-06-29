// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Resource {
    name: string;
    createdAt: string;
    modifiedAt: string;
    uuid: string;
    ownerUuid: string;
    href: string;
    kind: ResourceKind;
}

export enum ResourceKind {
    PROJECT = "project",
    COLLECTION = "collection",
    PIPELINE = "pipeline",
    UNKNOWN = "unknown"
}

export function getResourceKind(itemKind: string) {
    switch (itemKind) {
        case "arvados#project":
        case "arvados#group":
            return ResourceKind.PROJECT;
        case "arvados#collection":
            return ResourceKind.COLLECTION;
        case "arvados#pipeline":
            return ResourceKind.PIPELINE;
        default:
            return ResourceKind.UNKNOWN;
    }
}
