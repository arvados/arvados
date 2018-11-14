// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, TreeNode, setNode, Tree, TreeNodeStatus, setNodeStatus, expandNode, deactivateNode, deselectNode, selectNode, selectNodes, deselectNodes } from '~/models/tree';
import { TreePicker } from "./tree-picker";
import { treePickerActions, TreePickerAction } from "./tree-picker-actions";
import { compose } from "redux";
import { activateNode, getNode, toggleNodeCollapse, toggleNodeSelection } from '~/models/tree';
import { pipe } from 'lodash/fp';

export const treePickerReducer = (state: TreePicker = {}, action: TreePickerAction) =>
    treePickerActions.match(action, {
        LOAD_TREE_PICKER_NODE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, setNodeStatus(id)(TreeNodeStatus.PENDING)),

        LOAD_TREE_PICKER_NODE_SUCCESS: ({ id, nodes, pickerId }) =>
            updateOrCreatePicker(state, pickerId, compose(receiveNodes(nodes)(id), setNodeStatus(id)(TreeNodeStatus.LOADED))),

        TOGGLE_TREE_PICKER_NODE_COLLAPSE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, toggleNodeCollapse(id)),

        ACTIVATE_TREE_PICKER_NODE: ({ id, pickerId, relatedTreePickers = [] }) =>
            pipe(
                () => relatedTreePickers.reduce(
                    (state, relatedPickerId) => updateOrCreatePicker(state, relatedPickerId, deactivateNode),
                    state
                ),
                state => updateOrCreatePicker(state, pickerId, activateNode(id))
            )(),

        DEACTIVATE_TREE_PICKER_NODE: ({ pickerId }) =>
            updateOrCreatePicker(state, pickerId, deactivateNode),

        TOGGLE_TREE_PICKER_NODE_SELECTION: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, toggleNodeSelection(id)),

        SELECT_TREE_PICKER_NODE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, selectNodes(id)),

        DESELECT_TREE_PICKER_NODE: ({ id, pickerId }) =>
            updateOrCreatePicker(state, pickerId, deselectNodes(id)),

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
