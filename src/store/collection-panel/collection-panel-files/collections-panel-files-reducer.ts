// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionPanelFilesState, CollectionPanelFile, CollectionPanelDirectory, mapCollectionFileToCollectionPanelFile } from "./collection-panel-files-state";
import { CollectionPanelFilesAction, collectionPanelFilesAction } from "./collection-panel-files-actions";
import { createTree, mapTree, TreeNode, mapNodes, getNode, setNode, getNodeAncestors, getNodeDescendants, mapNodeValue } from "../../../models/tree";
import { CollectionFileType } from "../../../models/collection-file";

export const collectionPanelFilesReducer = (state: CollectionPanelFilesState = createTree(), action: CollectionPanelFilesAction) => {
    return collectionPanelFilesAction.match(action, {
        SET_COLLECTION_FILES: ({ files }) =>
            mapTree(mapCollectionFileToCollectionPanelFile)(files),

        TOGGLE_COLLECTION_FILE_COLLAPSE: data =>
            toggleCollapse(data.id)(state),

        TOGGLE_COLLECTION_FILE_SELECTION: data => [state]
            .map(toggleSelected(data.id))
            .map(toggleAncestors(data.id))
            .map(toggleDescendants(data.id))[0],

        SELECT_ALL_COLLECTION_FILES: () =>
            mapTree(mapNodeValue(v => ({ ...v, selected: true })))(state),

        UNSELECT_ALL_COLLECTION_FILES: () =>
            mapTree(mapNodeValue(v => ({ ...v, selected: false })))(state),
            
        default: () => state
    });
};

const toggleCollapse = (id: string) => (tree: CollectionPanelFilesState) =>
    mapNodes
        ([id])
        (mapNodeValue((v: CollectionPanelDirectory | CollectionPanelFile) =>
            v.type === CollectionFileType.DIRECTORY
                ? { ...v, collapsed: !v.collapsed }
                : v))
        (tree);

const toggleSelected = (id: string) => (tree: CollectionPanelFilesState) =>
    mapNodes
        ([id])
        (mapNodeValue((v: CollectionPanelDirectory | CollectionPanelFile) => ({ ...v, selected: !v.selected })))
        (tree);


const toggleDescendants = (id: string) => (tree: CollectionPanelFilesState) => {
    const node = getNode(id)(tree);
    if (node && node.value.type === CollectionFileType.DIRECTORY) {
        return mapNodes(getNodeDescendants(id)(tree))(mapNodeValue(v => ({ ...v, selected: node.value.selected })))(tree);
    }
    return tree;
};

const toggleAncestors = (id: string) => (tree: CollectionPanelFilesState) => {
    const ancestors = getNodeAncestors(id)(tree)
        .map(id => getNode(id)(tree))
        .reverse();
    return ancestors.reduce((newTree, parent) => parent !== undefined ? toggleParentNode(parent)(newTree) : newTree, tree);
};

const toggleParentNode = (node: TreeNode<CollectionPanelDirectory | CollectionPanelFile>) => (tree: CollectionPanelFilesState) => {
    const parentNode = getNode(node.id)(tree);
    if (parentNode) {
        const selected = parentNode.children
            .map(id => getNode(id)(tree))
            .every(node => node !== undefined && node.value.selected);
        return setNode(mapNodeValue(v => ({ ...v, selected }))(parentNode))(tree);
    }
    return setNode(node)(tree);
};


