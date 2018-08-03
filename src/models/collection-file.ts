// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree } from './tree';

export type CollectionFilesTree = Tree<CollectionDirectory | CollectionFile>;

export enum CollectionFileType {
    DIRECTORY = 'directory',
    FILE = 'file'
}

export interface CollectionDirectory {
    parentId: string;
    id: string;
    name: string;
    type: CollectionFileType.DIRECTORY;
}

export interface CollectionFile {
    parentId: string;
    id: string;
    name: string;
    size: number;
    type: CollectionFileType.FILE;
}

export const createCollectionDirectory = (data: Partial<CollectionDirectory>): CollectionDirectory => ({
    id: '',
    name: '',
    parentId: '',
    type: CollectionFileType.DIRECTORY,
    ...data
});

export const createCollectionFile = (data: Partial<CollectionFile>): CollectionFile => ({
    id: '',
    name: '',
    parentId: '',
    size: 0,
    type: CollectionFileType.FILE,
    ...data
});