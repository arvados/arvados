// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind, TrashResource } from "./resource";

export interface CollectionResource extends TrashResource {
    kind: ResourceKind.COLLECTION;
    name: string;
    description: string;
    properties: any;
    portableDataHash: string;
    manifestText: string;
    replicationDesired: number;
    replicationConfirmed: number;
    replicationConfirmedAt: string;
}

export const getCollectionUrl = (uuid: string) => {
    return `/collections/${uuid}`;
};
