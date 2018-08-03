// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { collectionPanelFilesReducer } from "./collections-panel-files-reducer";
import { collectionPanelFilesAction } from "./collection-panel-files-actions";
import { CollectionFile, CollectionDirectory, createCollectionFile, createCollectionDirectory } from "../../../models/collection-file";
import { createTree, setNode, getNodeValue, mapTreeValues, Tree } from "../../../models/tree";
import { CollectionPanelFile, CollectionPanelDirectory } from "./collection-panel-files-state";

describe('CollectionPanelFilesReducer', () => {

    const files: Array<CollectionFile | CollectionDirectory> = [
        createCollectionDirectory({ id: 'Directory 1', name: 'Directory 1', parentId: '' }),
        createCollectionDirectory({ id: 'Directory 2', name: 'Directory 2', parentId: 'Directory 1' }),
        createCollectionDirectory({ id: 'Directory 3', name: 'Directory 3', parentId: '' }),
        createCollectionDirectory({ id: 'Directory 4', name: 'Directory 4', parentId: 'Directory 3' }),
        createCollectionFile({ id: 'file1.txt', name: 'file1.txt', parentId: 'Directory 2' }),
        createCollectionFile({ id: 'file2.txt', name: 'file2.txt', parentId: 'Directory 2' }),
        createCollectionFile({ id: 'file3.txt', name: 'file3.txt', parentId: 'Directory 3' }),
        createCollectionFile({ id: 'file4.txt', name: 'file4.txt', parentId: 'Directory 3' }),
        createCollectionFile({ id: 'file5.txt', name: 'file5.txt', parentId: 'Directory 4' }),
    ];

    const collectionFilesTree = files.reduce((tree, file) => setNode({
        children: [],
        id: file.id,
        parent: file.parentId,
        value: file
    })(tree), createTree<CollectionFile | CollectionDirectory>());

    const collectionPanelFilesTree = collectionPanelFilesReducer(
        createTree<CollectionPanelFile | CollectionPanelDirectory>(),
        collectionPanelFilesAction.SET_COLLECTION_FILES({ files: collectionFilesTree }));

    it('SET_COLLECTION_FILES', () => {
        expect(getNodeValue('Directory 1')(collectionPanelFilesTree)).toEqual({
            ...createCollectionDirectory({ id: 'Directory 1', name: 'Directory 1', parentId: '' }),
            collapsed: true,
            selected: false
        });
    });

    it('TOGGLE_COLLECTION_FILE_COLLAPSE', () => {
        const newTree = collectionPanelFilesReducer(
            collectionPanelFilesTree,
            collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_COLLAPSE({ id: 'Directory 3' }));

        const value = getNodeValue('Directory 3')(newTree)! as CollectionPanelDirectory;
        expect(value.collapsed).toBe(false);
    });

    it('TOGGLE_COLLECTION_FILE_SELECTION', () => {
        const newTree = collectionPanelFilesReducer(
            collectionPanelFilesTree,
            collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: 'Directory 3' }));

        const value = getNodeValue('Directory 3')(newTree);
        expect(value!.selected).toBe(true);
    });

    it('TOGGLE_COLLECTION_FILE_SELECTION ancestors', () => {
        const newTree = collectionPanelFilesReducer(
            collectionPanelFilesTree,
            collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: 'Directory 2' }));

        const value = getNodeValue('Directory 1')(newTree);
        expect(value!.selected).toBe(true);
    });

    it('TOGGLE_COLLECTION_FILE_SELECTION descendants', () => {
        const newTree = collectionPanelFilesReducer(
            collectionPanelFilesTree,
            collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: 'Directory 2' }));
        expect(getNodeValue('file1.txt')(newTree)!.selected).toBe(true);
        expect(getNodeValue('file2.txt')(newTree)!.selected).toBe(true);
    });

    it('TOGGLE_COLLECTION_FILE_SELECTION unselect ancestors', () => {
        const [newTree] = [collectionPanelFilesTree]
            .map(tree => collectionPanelFilesReducer(
                tree,
                collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: 'Directory 2' })))
            .map(tree => collectionPanelFilesReducer(
                tree,
                collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: 'file1.txt' })));

        expect(getNodeValue('Directory 2')(newTree)!.selected).toBe(false);
    });

    it('SELECT_ALL_COLLECTION_FILES', () => {
        const newTree = collectionPanelFilesReducer(
            collectionPanelFilesTree,
            collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());

        mapTreeValues((v: CollectionPanelFile | CollectionPanelDirectory) => {
            expect(v.selected).toEqual(true);
            return v;
        })(newTree);
    });

    it('SELECT_ALL_COLLECTION_FILES', () => {
        const [newTree] = [collectionPanelFilesTree]
            .map(tree => collectionPanelFilesReducer(
                tree,
                collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES()))
            .map(tree => collectionPanelFilesReducer(
                tree,
                collectionPanelFilesAction.UNSELECT_ALL_COLLECTION_FILES()));

        mapTreeValues((v: CollectionPanelFile | CollectionPanelDirectory) => {
            expect(v.selected).toEqual(false);
            return v;
        })(newTree);
    });
});
