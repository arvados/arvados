// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, getNodeValue, getNodeChildren } from "../../models/tree";
import { TreePickerNode, createTreePickerNode } from "./tree-picker";
import { treePickerReducer } from "./tree-picker-reducer";
import { treePickerActions } from "./tree-picker-actions";
import { TreeItemStatus } from "../../components/tree/tree";


describe('TreePickerReducer', () => {
    it('LOAD_TREE_PICKER_NODE - initial state', () => {
        const tree = createTree<TreePickerNode>();
        const newTree = treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1' }));
        expect(newTree).toEqual(tree);
    });

    it('LOAD_TREE_PICKER_NODE', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE({ id: '1' })));
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            status: TreeItemStatus.PENDING
        });
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS - initial state', () => {
        const tree = createTree<TreePickerNode>();
        const subNode = createTreePickerNode({ id: '1.1', value: '1.1' });
        const newTree = treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [subNode] }));
        expect(getNodeChildren('')(newTree)).toEqual(['1.1']);
    });

    it('LOAD_TREE_PICKER_NODE_SUCCESS', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const subNode = createTreePickerNode({ id: '1.1', value: '1.1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '1', nodes: [subNode] })));
        expect(getNodeChildren('1')(newTree)).toEqual(['1.1']);
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            status: TreeItemStatus.LOADED
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - collapsed', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1' })));
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            collapsed: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_COLLAPSE - expanded', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1' })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id: '1' })));
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            collapsed: false
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_SELECT - selected', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ id: '1' })));
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            selected: true
        });
    });

    it('TOGGLE_TREE_PICKER_NODE_SELECT - not selected', () => {
        const tree = createTree<TreePickerNode>();
        const node = createTreePickerNode({ id: '1', value: '1' });
        const [newTree] = [tree]
            .map(tree => treePickerReducer(tree, treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({ id: '', nodes: [node] })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ id: '1' })))
            .map(tree => treePickerReducer(tree, treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ id: '1' })));
        expect(getNodeValue('1')(newTree)).toEqual({
            ...createTreePickerNode({ id: '1', value: '1' }),
            selected: false
        });
    });
});
