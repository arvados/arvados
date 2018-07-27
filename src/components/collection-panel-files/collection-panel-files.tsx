// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { FileTreeData } from '../file-tree/file-tree-data';
import { FileTree } from '../file-tree/file-tree';
import { CollectionPanelFilesState } from '../../store/collection-panel/collection-panel-files/collection-panel-files-state';

export interface CollectionPanelFilesProps {
    items: Array<TreeItem<FileTreeData>>;
    onItemContextMenu: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCommonContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
}

export const CollectionPanelFiles = ({ onItemContextMenu, onCommonContextMenu, ...treeProps }: CollectionPanelFilesProps) =>
    <div>
        <FileTree onContextMenu={onItemContextMenu} {...treeProps} />
    </div>;

export const collectionPanelItems: Array<TreeItem<FileTreeData>> = [{
    active: false,
    data: {
        name: "Directory 1",
        type: "directory"
    },
    id: "Directory 1",
    open: true,
    status: TreeItemStatus.LOADED,
    items: [{
        active: false,
        data: {
            name: "Directory 1.1",
            type: "directory"
        },
        id: "Directory 1.1",
        open: false,
        status: TreeItemStatus.LOADED,
        items: []
    }, {
        active: false,
        data: {
            name: "File 1.1",
            type: "file"
        },
        id: "File 1.1",
        open: false,
        status: TreeItemStatus.LOADED,
        items: []
    }]
}, {
    active: false,
    data: {
        name: "Directory 2",
        type: "directory"
    },
    id: "Directory 2",
    open: false,
    status: TreeItemStatus.LOADED
}]; 
