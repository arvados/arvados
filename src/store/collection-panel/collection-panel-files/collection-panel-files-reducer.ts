// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionPanelFilesState, CollectionPanelFile, CollectionPanelDirectory, mapCollectionFileToCollectionPanelFile, mergeCollectionPanelFilesStates } from './collection-panel-files-state';
import { CollectionPanelFilesAction, collectionPanelFilesAction } from "./collection-panel-files-actions";
import { createTree, mapTreeValues, getNode, setNode, getNodeAncestorsIds, getNodeDescendantsIds, setNodeValueWith, mapTree } from "models/tree";
import { CollectionFileType } from "models/collection-file";

let fetchedFiles: any = {};

export const collectionPanelFilesReducer = (state: CollectionPanelFilesState = createTree(), action: CollectionPanelFilesAction) => {
    // Low-level tree handling setNode() func does in-place data modifications
    // for performance reasons, so we pass a copy of 'state' to avoid side effects.
    return collectionPanelFilesAction.match(action, {
        SET_COLLECTION_FILES: files => {
            fetchedFiles = files;
            return mergeCollectionPanelFilesStates({ ...state }, mapTree(mapCollectionFileToCollectionPanelFile)(files));
        },

        TOGGLE_COLLECTION_FILE_COLLAPSE: data =>
            toggleCollapse(data.id)({ ...state }),

        TOGGLE_COLLECTION_FILE_SELECTION: data => [{ ...state }]
            .map(toggleSelected(data.id))
            .map(toggleAncestors(data.id))
            .map(toggleDescendants(data.id))[0],

        ON_SEARCH_CHANGE: (searchValue) => {
            const fileIds: string[] = [];
            const directoryIds: string[] = [];
            const filteredFiles = Object.keys(fetchedFiles)
                .filter((key: string) => {
                    const node = fetchedFiles[key];

                    if (node.value === undefined) {
                        return false;
                    }

                    const { id, value: { type, name } } = node;

                    if (type === CollectionFileType.DIRECTORY) {
                        directoryIds.push(id);
                        return true;
                    }

                    const includeFile = name.toLowerCase().indexOf(searchValue.toLowerCase()) > -1;

                    if (includeFile) {
                        fileIds.push(id);
                    }

                    return includeFile;
                })
                .reduce((prev, next) => {
                    const node = JSON.parse(JSON.stringify(fetchedFiles[next]));
                    const { value: { type }, children } = node;

                    node.children = node.children.filter((key: string) => {
                        const isFile = directoryIds.indexOf(key) === -1;
                        return isFile ?
                          fileIds.indexOf(key) > -1 :
                          !!fileIds.find(id => id.indexOf(key) > -1);
                    });

                    if (type === CollectionFileType.FILE || children.length > 0) {
                        prev[next] = node;
                    }

                    return prev;
                }, {});

            return mapTreeValues((v: CollectionPanelDirectory | CollectionPanelFile) => {
                if (v.type === CollectionFileType.DIRECTORY) {
                    return ({ 
                        ...v,
                        collapsed: searchValue.length === 0,
                    });
                }

                return ({ ...v });
            })({ ...filteredFiles });
        },

        SELECT_ALL_COLLECTION_FILES: () =>
            mapTreeValues(v => ({ ...v, selected: true }))({ ...state }),

        UNSELECT_ALL_COLLECTION_FILES: () =>
            mapTreeValues(v => ({ ...v, selected: false }))({ ...state }),

        default: () => state
    }) as CollectionPanelFilesState;
};

const toggleCollapse = (id: string) => (tree: CollectionPanelFilesState) =>
    setNodeValueWith((v: CollectionPanelDirectory | CollectionPanelFile) =>
        v.type === CollectionFileType.DIRECTORY
            ? { ...v, collapsed: !v.collapsed }
            : v)(id)(tree);


const toggleSelected = (id: string) => (tree: CollectionPanelFilesState) =>
    setNodeValueWith((v: CollectionPanelDirectory | CollectionPanelFile) => ({ ...v, selected: !v.selected }))(id)(tree);


const toggleDescendants = (id: string) => (tree: CollectionPanelFilesState) => {
    const node = getNode(id)(tree);
    if (node && node.value.type === CollectionFileType.DIRECTORY) {
        return getNodeDescendantsIds(id)(tree)
            .reduce((newTree, id) =>
                setNodeValueWith(v => ({ ...v, selected: node.value.selected }))(id)(newTree), tree);
    }
    return tree;
};

const toggleAncestors = (id: string) => (tree: CollectionPanelFilesState) => {
    const ancestors = getNodeAncestorsIds(id)(tree).reverse();
    return ancestors.reduce((newTree, parent) => parent ? toggleParentNode(parent)(newTree) : newTree, tree);
};

const toggleParentNode = (id: string) => (tree: CollectionPanelFilesState) => {
    const node = getNode(id)(tree);
    if (node) {
        const parentNode = getNode(node.id)(tree);
        if (parentNode) {
            const selected = parentNode.children
                .map(id => getNode(id)(tree))
                .every(node => node !== undefined && node.value.selected);
            return setNodeValueWith(v => ({ ...v, selected }))(parentNode.id)(tree);
        }
        return setNode(node)(tree);
    }
    return tree;
};


