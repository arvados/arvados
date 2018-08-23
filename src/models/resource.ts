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
    COLLECTION = "arvados#collection",
    CONTAINER_REQUEST = "arvados#containerRequest",
    GROUP = "arvados#group",
    PROCESS = "arvados#containerRequest",
    PROJECT = "arvados#group",
    WORKFLOW = "arvados#workflow",
    USER = "arvados#user",
}

export enum ResourceObjectType {
    USER = 'tpzed',
    GROUP = 'j7d0g',
    COLLECTION = '4zz18'
}

export const extractUuidObjectType = (uuid: string) => {
    const match = /(.{5})-(.{5})-(.{15})/.exec(uuid);
    return match
        ? match[2]
        : undefined;
};

export const extractUuidKind = (uuid: string = '') => {
    const objectType = extractUuidObjectType(uuid);
    switch(objectType){
        case ResourceObjectType.USER:
            return ResourceKind.USER;
        case ResourceObjectType.GROUP:
            return ResourceKind.GROUP;
        case ResourceObjectType.COLLECTION:
            return ResourceKind.COLLECTION;
        default:
            return undefined;
    }
};
