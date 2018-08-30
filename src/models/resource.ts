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
    CONTAINER = "arvados#container",
    CONTAINER_REQUEST = "arvados#containerRequest",
    GROUP = "arvados#group",
    PROCESS = "arvados#containerRequest",
    PROJECT = "arvados#group",
    USER = "arvados#user",
    WORKFLOW = "arvados#workflow",
}

export enum ResourceObjectType {
    COLLECTION = '4zz18',
    CONTAINER = 'dz642',
    CONTAINER_REQUEST = 'xvhdp',
    GROUP = 'j7d0g',
    USER = 'tpzed',
}

export const RESOURCE_UUID_PATTERN = '.{5}-.{5}-.{15}';
export const RESOURCE_UUID_REGEX = new RegExp(RESOURCE_UUID_PATTERN);

export const isResourceUuid = (uuid: string) =>
    RESOURCE_UUID_REGEX.test(uuid);

export const extractUuidObjectType = (uuid: string) => {
    const match = RESOURCE_UUID_REGEX.exec(uuid);
    return match
        ? match[0].split('-')[1]
        : undefined;
};

export const extractUuidKind = (uuid: string = '') => {
    const objectType = extractUuidObjectType(uuid);
    switch (objectType) {
        case ResourceObjectType.USER:
            return ResourceKind.USER;
        case ResourceObjectType.GROUP:
            return ResourceKind.GROUP;
        case ResourceObjectType.COLLECTION:
            return ResourceKind.COLLECTION;
        case ResourceObjectType.CONTAINER_REQUEST:
            return ResourceKind.CONTAINER_REQUEST;
        case ResourceObjectType.CONTAINER:
            return ResourceKind.CONTAINER;
        default:
            return undefined;
    }
};
