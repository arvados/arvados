// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind, TrashableResource } from "./resource";

export interface CollectionResource extends TrashableResource {
    kind: ResourceKind.COLLECTION;
    name: string;
    description: string;
    properties: any;
    portableDataHash: string;
    manifestText: string;
    replicationDesired: number;
    replicationConfirmed: number;
    replicationConfirmedAt: string;
    storageClassesDesired: string[];
    storageClassesConfirmed: string[];
    storageClassesConfirmedAt: string;
}

export const getCollectionUrl = (uuid: string) => {
    return `/collections/${uuid}`;
};

export enum CollectionType {
    GENERAL = 'nil',
    OUTPUT = 'output',
    LOG = 'log',
}
