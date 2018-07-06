// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum ResourceKind {
    Collection = "arvados#collection",
    ContainerRequest = "arvados#containerRequest",
    Group = "arvados#group",
    Process = "arvados#containerRequest",
    Project = "arvados#group",
    Workflow = "arvados#workflow"
}