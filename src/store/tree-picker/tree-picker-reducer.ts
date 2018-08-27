// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, setNodeValueWith, TreeNode, setNode, mapTreeValues, Tree } from "~/models/tree";
import { TreePicker, TreePickerNode } from "./tree-picker";
import { treePickerActions, TreePickerAction } from "./tree-picker-actions";
import { TreeItemStatus } from "~/components/tree/tree";
import { compose } from "redux";

export const treePickerReducer = (state: TreePicker = {}, action: TreePickerAction) =>
    treePickerActions.match(action, {
        LOAD_TREE_PICKER_NODE: ({ nodeId, pickerId }) =>
            updateOrCreatePicker(state, pickerId, setNodeValueWith(setPending)(nodeId)),
        LOAD_TREE_PICKER_NODE_SUCCESS: ({ nodeId, nodes, pickerId }) =>
            updateOrCreatePicker(state, pickerId, compose(receiveNodes(nodes)(nodeId), setNodeValueWith(setLoaded)(nodeId))),
        TOGGLE_TREE_PICKER_NODE_COLLAPSE: ({ nodeId, pickerId }) =>
            updateOrCreatePicker(state, pickerId, setNodeValueWith(toggleCollapse)(nodeId)),
        TOGGLE_TREE_PICKER_NODE_SELECT: ({ nodeId, pickerId }) =>
            updateOrCreatePicker(state, pickerId, mapTreeValues(toggleSelect(nodeId))),
        RESET_TREE_PICKER: ({ pickerId }) =>
            updateOrCreatePicker(state, pickerId, createTree),
        EXPAND_TREE_PICKER_NODES: ({ pickerId, nodeIds }) =>
            updateOrCreatePicker(state, pickerId, mapTreeValues(expand(nodeIds))),
        default: () => state
    });

const updateOrCreatePicker = (state: TreePicker, pickerId: string, func: (value: Tree<TreePickerNode>) => Tree<TreePickerNode>) => {
    const picker = state[pickerId] || createTree();
    const updatedPicker = func(picker);
    return { ...state, [pickerId]: updatedPicker };
};

const expand = (ids: string[]) => (node: TreePickerNode): TreePickerNode =>
    ids.some(id => id === node.nodeId)
        ? { ...node, collapsed: false }
        : node;

const setPending = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, status: TreeItemStatus.PENDING });

const setLoaded = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, status: TreeItemStatus.LOADED });

const toggleCollapse = (value: TreePickerNode): TreePickerNode =>
    ({ ...value, collapsed: !value.collapsed });

const toggleSelect = (nodeId: string) => (value: TreePickerNode): TreePickerNode =>
    value.nodeId === nodeId
        ? ({ ...value, selected: !value.selected })
        : ({ ...value, selected: false });

const receiveNodes = (nodes: Array<TreePickerNode>) => (parent: string) => (state: Tree<TreePickerNode>) =>
    nodes.reduce((tree, node) =>
        setNode(
            createTreeNode(parent)(node)
        )(tree), state);

const createTreeNode = (parent: string) => (node: TreePickerNode): TreeNode<TreePickerNode> => ({
    children: [],
    id: node.nodeId,
    parent,
    value: node
});
