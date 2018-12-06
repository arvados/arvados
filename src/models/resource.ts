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
    kind: ResourceKind;
    etag: string;
}

export interface TrashableResource extends Resource {
    trashAt: string;
    deleteAt: string;
    isTrashed: boolean;
}

export enum ResourceKind {
    API_CLIENT_AUTHORIZATION = "arvados#apiClientAuthorization",
    COLLECTION = "arvados#collection",
    CONTAINER = "arvados#container",
    CONTAINER_REQUEST = "arvados#containerRequest",
    GROUP = "arvados#group",
    LOG = "arvados#log",
    NODE = "arvados#node",
    PROCESS = "arvados#containerRequest",
    PROJECT = "arvados#group",
    REPOSITORY = "arvados#repository",
    SSH_KEY = "arvados#authorizedKeys",
    KEEP_SERVICE = "arvados#keepService",
    USER = "arvados#user",
    VIRTUAL_MACHINE = "arvados#virtualMachine",
    WORKFLOW = "arvados#workflow",
    NONE = "arvados#none"
}

export enum ResourceObjectType {
    API_CLIENT_AUTHORIZATION = 'gj3su',
    COLLECTION = '4zz18',
    CONTAINER = 'dz642',
    CONTAINER_REQUEST = 'xvhdp',
    GROUP = 'j7d0g',
    LOG = '57u5n',
    REPOSITORY = 's0uqq',
    USER = 'tpzed',
    VIRTUAL_MACHINE = '2x53u',
    WORKFLOW = '7fd4e',
    SSH_KEY = 'fngyi',
    KEEP_SERVICE = 'bi6l4',
    NODE = '7ekkf'
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
        case ResourceObjectType.LOG:
            return ResourceKind.LOG;
        case ResourceObjectType.WORKFLOW:
            return ResourceKind.WORKFLOW;
        case ResourceObjectType.VIRTUAL_MACHINE:
            return ResourceKind.VIRTUAL_MACHINE;
        case ResourceObjectType.REPOSITORY:
            return ResourceKind.REPOSITORY;
        case ResourceObjectType.SSH_KEY:
            return ResourceKind.SSH_KEY;
        case ResourceObjectType.KEEP_SERVICE:
            return ResourceKind.KEEP_SERVICE;
        case ResourceObjectType.NODE:
            return ResourceKind.NODE;
        case ResourceObjectType.API_CLIENT_AUTHORIZATION:
            return ResourceKind.API_CLIENT_AUTHORIZATION;
        default:
            return undefined;
    }
};
