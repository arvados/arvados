// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Resource {
    uuid: string;
    ownerUuid: string;
    createdAt: string;
    modifiedByClientUuid: string;
    modifiedByUserUuid: string;
    modifiedAt: string;
    href: string;
    kind: string;
    etag: string;
}

export interface TrashResource extends Resource {
    trashAt: string;
    deleteAt: string;
    isTrashed: boolean;
}

export enum ResourceKind {
    COLLECTION = "arvados#collection",
    GROUP = "arvados#group",
    PROCESS = "arvados#containerRequest",
    PROJECT = "arvados#group",
    WORKFLOW = "arvados#workflow"
}
