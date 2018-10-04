// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, getNodeValue, getNodeChildrenIds } from "~/models/tree";
import { treePickerReducer } from "./tree-picker-reducer";
import { treePickerActions } from "./tree-picker-actions";
import { TreeItemStatus } from "~/components/tree/tree";
import { initTreeNode } from '~/models/tree';

describe('TreePickerReducer', () => {
    it('LOAD_TREE_PICKER_NODE - initial state', () => {
        const tree = createTree<{}>();
        const newState = treePickerReducer({}, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1', pickerId: "projects" }));
        expect(newState).toEqual({ 'projects': tree });
    });

    it('LOAD_TREE_PICKER_NODE', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1', pickerId: "projects" })));

        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            status: TreeItemStatus.PENDING
        });
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS - initial state', () => {
        const subNode = initTreeNode({ id: '1.1', value: '1.1' });
        const newState = treePickerReducer({}, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [subNode], pickerId: "projects" }));
        expect(getNodeChildrenIds('')(newState.projects)).toEqual(['1.1']);
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const subNode = initTreeNode({ id: '1.1', value: '1.1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '1', nodes: [subNode], pickerId: "projects" })));
        expect(getNodeChildrenIds('1')(newState.projects)).toEqual(['1.1']);
        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            status: TreeItemStatus.LOADED
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - collapsed', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" })));
        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            collapsed: false
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - expanded', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1', pickerId: "projects" })));
        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            collapsed: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_SELECT - selected', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: '1', pickerId: "projects" })));
        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            selected: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_SELECT - not selected', () => {
        const node = initTreeNode({ id: '1', value: '1' });
        const [newState] = [{
            projects: createTree<{}>()
        }]
            .map(state => treePickerReducer(state, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node], pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: '1', pickerId: "projects" })))
            .map(state => treePickerReducer(state, treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: '1', pickerId: "projects" })));
        expect(getNodeValue('1')(newState.projects)).toEqual({
            ...initTreeNode({ id: '1', value: '1' }),
            selected: false
        });
    });
});
