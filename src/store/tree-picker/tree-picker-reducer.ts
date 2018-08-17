// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, setNodeValueWith, TreeNode, setNode, mapTreeValues, Tree } from "~/models/tree";
import { TreePicker, TreePickerNode } from "./tree-picker";
import { treePickerActions, TreePickerAction } from "./tree-picker-actions";
import { TreeItemStatus } from "~/components/tree/tree";

export const treePickerReducer = (state: TreePicker = {}, action: TreePickerAction) =>
    treePickerActions.match(action, {
        LOAD_TREE_PICKER_NODE: ({ id, pickerId }) => {
            const picker = state[pickerId] || createTree();
            const updatedPicker = setNodeValueWith(setPending)(id)(picker);
            return { ...state, [pickerId]: updatedPicker };
        },
        LOAD_TREE_PICKER_NODE_SUCCESS: ({ id, nodes, pickerId }) => {
            const picker = state[pickerId] || createTree();
            const [updatedPicker] = [picker]
                .map(receiveNodes(nodes)(id))
                .map(setNodeValueWith(setLoaded)(id));
            return { ...state, [pickerId]: updatedPicker };
        },
        TOGGLE_TREE_PICKER_NODE_COLLAPSE: ({ id, pickerId }) => {
            const picker = state[pickerId] || createTree();
            const updatedPicker = setNodeValueWith(toggleCollapse)(id)(picker);
            return { ...state, [pickerId]: updatedPicker };
        },
        TOGGLE_TREE_PICKER_NODE_SELECT: ({ id, pickerId }) => {
            const picker = state[pickerId] || createTree();
            const updatedPicker = mapTreeValues(toggleSelect(id))(picker);
            return { ...state, [pickerId]: updatedPicker };
        },
        default: () => state
    });

const setPending = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, status: TreeItemStatus.PENDING });

const setLoaded = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, status: TreeItemStatus.LOADED });

const toggleCollapse = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, collapsed: !value.collapsed });

const toggleSelect = (id: string) => (value: TreePickerNode): TreePickerNode =>
    value.id === id
        ? ({ ...value, selected: !value.selected })
        : ({ ...value, selected: false });

const receiveNodes = (nodes: Array<TreePickerNode>) => (parent: string) => (state: Tree<TreePickerNode>) =>
    nodes.reduce((tree, node) => 
        setNode(
            createTreeNode(parent)(node)
        )(tree), state);

const createTreeNode = (parent: string) => (node: TreePickerNode): TreeNode<TreePickerNode> => ({
    children: [],
    id: node.id,
    parent,
    value: node
});
