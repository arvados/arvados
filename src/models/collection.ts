// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource as R } from "./resource";
import { Resource } from "../common/api/common-resource-service";
import { ResourceKind } from "./kinds";

export interface Collection extends R {
}

export interface CollectionResource extends Resource {
    kind: ResourceKind.Collection;
    name: string;
    description: string;
    properties: any;
    portableDataHash: string;
    manifestText: string;
    replicationDesired: number;
    replicationConfirmed: number;
    replicationConfirmedAt: string;
    trashAt: string;
    deleteAt: string;
    isTrashed: boolean;
}
