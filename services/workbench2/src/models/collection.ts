// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    Resource,
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

// We exclude 'manifestText' and 'unsignedManifestText' from the default
export const defaultCollectionSelectedFields = [
    'name',
    'description',
    'portableDataHash',
    'replicationDesired',
    'replicationConfirmed',
    'replicationConfirmedAt',
    'storageClassesDesired',
    'storageClassesConfirmed',
    'storageClassesConfirmedAt',
    'currentVersionUuid',
    'version',
    'preserveVersion',
    'fileCount',
    'fileSizeTotal',
    // ResourceWithProperties field
    'properties',
    // TrashableResource fields
    'trashAt',
    'deleteAt',
    'isTrashed',
    // Resource fields
    'uuid',
    'ownerUuid',
    'createdAt',
    'modifiedByUserUuid',
    'modifiedAt',
    'kind',
    'etag',
];

export const getCollectionUrl = (uuid: string) => {
    return `/collections/${uuid}`;
};

export const isCollectionResource = (resource?: Resource): resource is CollectionResource => {
    return !!resource && resource.kind === ResourceKind.COLLECTION;
};

export const isCollectionResourceLatestVersion = (resource?: Resource): boolean => {
    return isCollectionResource(resource) && resource.uuid === resource.currentVersionUuid;
};

export enum CollectionType {
    GENERAL = 'nil',
    OUTPUT = 'output',
    LOG = 'log',
    INTERMEDIATE = 'intermediate',
}
