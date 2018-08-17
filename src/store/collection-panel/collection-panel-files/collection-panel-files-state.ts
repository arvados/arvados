// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionFile, CollectionDirectory, CollectionFileType } from '~/models/collection-file';
import { Tree, TreeNode } from '~/models/tree';

export type CollectionPanelFilesState = Tree<CollectionPanelDirectory | CollectionPanelFile>;

export interface CollectionPanelDirectory extends CollectionDirectory {
    collapsed: boolean;
    selected: boolean;
}

export interface CollectionPanelFile extends CollectionFile {
    selected: boolean;
}

export const mapCollectionFileToCollectionPanelFile = (node: TreeNode<CollectionDirectory | CollectionFile>): TreeNode<CollectionPanelDirectory | CollectionPanelFile> => {
    return {
        ...node,
        value: node.value.type === CollectionFileType.DIRECTORY
            ? { ...node.value, selected: false, collapsed: true }
            : { ...node.value, selected: false }
    };
};
