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

export enum ResourceKind {
    Collection = "arvados#collection",
    ContainerRequest = "arvados#containerRequest",
    Group = "arvados#group",
    Process = "arvados#containerRequest",
    Project = "arvados#group",
    Workflow = "arvados#workflow"
}
