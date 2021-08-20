// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree, createTree, setNode, TreeNodeStatus } from './tree';
import { head, split, pipe, join } from 'lodash/fp';

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

export const createCollectionFilesTree = (data: Array<CollectionDirectory | CollectionFile>, joinParents: Boolean = true) => {
    const directories = data.filter(item => item.type === CollectionFileType.DIRECTORY);
    directories.sort((a, b) => a.path.localeCompare(b.path));
    const files = data.filter(item => item.type === CollectionFileType.FILE);
    return [...directories, ...files]
        .reduce((tree, item) => setNode({
            children: [],
            id: item.id,
            parent: joinParents ? getParentId(item) : '',
            value: item,
            active: false,
            selected: false,
            expanded: false,
            status: TreeNodeStatus.INITIAL
        })(tree), createTree<CollectionDirectory | CollectionFile>());
};

const getParentId = (item: CollectionDirectory | CollectionFile) =>
    item.path
        ? join('', [getCollectionId(item.id), item.path])
        : item.path;

const getCollectionId = pipe(
    split('/'),
    head,
);