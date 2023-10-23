// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export enum MountKind {
    COLLECTION = 'collection',
    GIT_TREE = 'git_tree',
    TEMPORARY_DIRECTORY = 'tmp',
    KEEP = 'keep',
    MOUNTED_FILE = 'file',
    JSON = 'json'
}

export type MountType =
    CollectionMount |
    GitTreeMount |
    TemporaryDirectoryMount |
    KeepMount |
    JSONMount |
    FileMount;

export interface CollectionMount {
    kind: MountKind.COLLECTION;
    uuid?: string;
    portable_data_hash?: string;
    path?: string;
    writable?: boolean;
}

export interface GitTreeMount {
    kind: MountKind.GIT_TREE;
    uuid: string;
    commit: string;
    path?: string;
}

export enum TemporaryDirectoryDeviceType {
    RAM = 'ram',
    SSD = 'ssd',
    DISK = 'disk',
    NETWORK = 'network',
}

export interface TemporaryDirectoryMount {
    kind: MountKind.TEMPORARY_DIRECTORY;
    capacity: number;
    deviceType: TemporaryDirectoryDeviceType;
}

export interface KeepMount {
    kind: MountKind.KEEP;
}

export interface JSONMount {
    kind: MountKind.JSON;
    content: any;
}

export interface FileMount {
    kind: MountKind.MOUNTED_FILE;
    path: string;
}
