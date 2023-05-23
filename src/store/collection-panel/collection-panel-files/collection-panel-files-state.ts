// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree, TreeNode, mapTreeValues, getNodeValue, getNodeDescendants } from 'models/tree';
import { CollectionFile, CollectionDirectory, CollectionFileType } from 'models/collection-file';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { CollectionResource } from 'models/collection';

export type CollectionPanelFilesState = Tree<CollectionPanelDirectory | CollectionPanelFile>;

export interface CollectionPanelDirectory extends CollectionDirectory {
    collapsed: boolean;
    selected: boolean;
}

export interface CollectionPanelFile extends CollectionFile {
    selected: boolean;
}

export interface CollectionFileSelection {
    collection: CollectionResource;
    selectedPaths: string[];
}

export const mapCollectionFileToCollectionPanelFile = (node: TreeNode<CollectionDirectory | CollectionFile>): TreeNode<CollectionPanelDirectory | CollectionPanelFile> => {
    return {
        ...node,
        value: node.value.type === CollectionFileType.DIRECTORY
            ? { ...node.value, selected: false, collapsed: true }
            : { ...node.value, selected: false }
    };
};

export const mergeCollectionPanelFilesStates = (oldState: CollectionPanelFilesState, newState: CollectionPanelFilesState) => {
    return mapTreeValues((value: CollectionPanelDirectory | CollectionPanelFile) => {
        const oldValue = getNodeValue(value.id)(oldState);
        return oldValue
            ? oldValue.type === CollectionFileType.DIRECTORY
                ? { ...value, collapsed: oldValue.collapsed, selected: oldValue.selected }
                : { ...value, selected: oldValue.selected }
            : value;
    })(newState);
};

export const filterCollectionFilesBySelection = (tree: CollectionPanelFilesState, selected: boolean): (CollectionPanelFile | CollectionPanelDirectory)[] => {
    const allFiles = getNodeDescendants('')(tree).map(node => node.value);
    const selectedDirectories = allFiles.filter(file => file.selected === selected && file.type === CollectionFileType.DIRECTORY);
    const selectedFiles = allFiles.filter(file => file.selected === selected && !selectedDirectories.some(dir => dir.id === file.path));
    return [...selectedDirectories, ...selectedFiles]
        .filter((value, index, array) => (
            array.indexOf(value) === index
        ));
};

export const getCollectionSelection = (sourceCollection: CollectionResource, selectedItems: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)[]) => ({
    collection: sourceCollection,
    selectedPaths: selectedItems.map(itemsToPaths).map(trimPathUuids(sourceCollection.uuid)),
})

const itemsToPaths = (item: (CollectionPanelDirectory | CollectionPanelFile | ContextMenuResource)): string => ('uuid' in item) ? item.uuid : item.id;

const trimPathUuids = (parentCollectionUuid: string) => (path: string) => path.replace(new RegExp(`(^${parentCollectionUuid})`), '');
