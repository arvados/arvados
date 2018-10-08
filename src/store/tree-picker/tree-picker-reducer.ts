// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, TreeNode, setNode, Tree, TreeNodeStatus, setNodeStatus, expandNode } from '~/models/tree';
import { TreePicker } from "./tree-picker";
import { treePickerActions, TreePickerAction } from "./tree-picker-actions";
import { compose } from "redux";
import { activateNode, getNode, toggleNodeCollapse, toggleNodeSelection } from '~/models/tree';

export const treePickerReducer = (state: TreePicker = {}, action: TreePickerAction) =>
    treePickerActions.match(action, {
        LOAD_TREE_PICKER_NODE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, setNodeStatus(id)(TreeNodeStatus.PENDING)),
        LOAD_TREE_PICKER_NODE_SUCCESS: ({ id, nodes, pickerId }) =>
            updateOrCreatePicker(state, pickerId, compose(receiveNodes(nodes)(id), setNodeStatus(id)(TreeNodeStatus.LOADED))),
        TOGGLE_TREE_PICKER_NODE_COLLAPSE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, toggleNodeCollapse(id)),
        ACTIVATE_TREE_PICKER_NODE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, activateNode(id)),
        TOGGLE_TREE_PICKER_NODE_SELECTION: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, toggleNodeSelection(id)),
        RESET_TREE_PICKER: ({ pickerId }) =>
            updateOrCreatePicker(state, pickerId, createTree),
        EXPAND_TREE_PICKER_NODES: ({ pickerId, ids }) =>
            updateOrCreatePicker(state, pickerId, expandNode(...ids)),
        default: () => state
    });

const updateOrCreatePicker = <V>(state: TreePicker, pickerId: string, func: (value: Tree<V>) => Tree<V>) => {
    const picker = state[pickerId] || createTree();
    const updatedPicker = func(picker);
    return { ...state, [pickerId]: updatedPicker };
};

const receiveNodes = <V>(nodes: Array<TreeNode<V>>) => (parent: string) => (state: Tree<V>) => {
    const parentNode = getNode(parent)(state);
    let newState = state;
    if (parentNode) {
        newState = setNode({ ...parentNode, children: [] })(state);
    }
    return nodes.reduce((tree, node) => {
        return setNode({ ...node, parent })(tree);
    }, newState);
};
