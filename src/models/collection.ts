// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    ResourceKind,
    TrashableResource,
    ResourceWithProperties
} from "./resource";

export interface CollectionResource extends TrashableResource, ResourceWithProperties {
    kind: ResourceKind.COLLECTION;
    name: string;
    description: string;
    portableDataHash: string;
    manifestText: string;
    replicationDesired: number;
    replicationConfirmed: number;
    replicationConfirmedAt: string;
    storageClassesDesired: string[];
    storageClassesConfirmed: string[];
    storageClassesConfirmedAt: string;
    currentVersionUuid: string;
    version: number;
    preserveVersion: boolean;
    unsignedManifestText?: string;
    fileCount: number;
    fileSizeTotal: number;
}

export const getCollectionUrl = (uuid: string) => {
    return `/collections/${uuid}`;
};

export enum CollectionType {
    GENERAL = 'nil',
    OUTPUT = 'output',
    LOG = 'log',
}
