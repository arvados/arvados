// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree, createTree, setNode } from './tree';

export type CollectionFilesTree = Tree<CollectionDirectory | CollectionFile>;

export enum CollectionFileType {
    DIRECTORY = 'directory',
    FILE = 'file'
}

export interface CollectionDirectory {
    path: string;
    url: string;
    id: string;
    name: string;
    type: CollectionFileType.DIRECTORY;
}

export interface CollectionFile {
    path: string;
    url: string;
    id: string;
    name: string;
    size: number;
    type: CollectionFileType.FILE;
}

export interface CollectionUploadFile {
    name: string;
}

export const createCollectionDirectory = (data: Partial<CollectionDirectory>): CollectionDirectory => ({
    id: '',
    name: '',
    path: '',
    url: '',
    type: CollectionFileType.DIRECTORY,
    ...data
});

export const createCollectionFile = (data: Partial<CollectionFile>): CollectionFile => ({
    id: '',
    name: '',
    path: '',
    url: '',
    size: 0,
    type: CollectionFileType.FILE,
    ...data
});

export const createCollectionFilesTree = (data: Array<CollectionDirectory | CollectionFile>) => {
    const directories = data.filter(item => item.type === CollectionFileType.DIRECTORY);
    directories.sort((a, b) => a.path.localeCompare(b.path));
    const files = data.filter(item => item.type === CollectionFileType.FILE);
    return [...directories, ...files]
        .reduce((tree, item) => setNode({
            children: [],
            id: item.id,
            parent: item.path,
            value: item
        })(tree), createTree<CollectionDirectory | CollectionFile>());
};