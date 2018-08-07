// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createTree, setNodeValueWith, TreeNode, setNode, mapTree, mapTreeValues } from "../../models/tree";
import { TreePicker, TreePickerNode } from "./tree-picker";
import { treePickerActions, TreePickerAction } from "./tree-picker-actions";
import { TreeItemStatus } from "../../components/tree/tree";


export const treePickerReducer = (state: TreePicker = createTree(), action: TreePickerAction) =>
    treePickerActions.match(action, {
        LOAD_TREE_PICKER_NODE: ({ id }) =>
            setNodeValueWith(setPending)(id)(state),
        LOAD_TREE_PICKER_NODE_SUCCESS: ({ id, nodes }) => {
            const [newState] = [state]
                .map(receiveNodes(nodes)(id))
                .map(setNodeValueWith(setLoaded)(id));
            return newState;
        },
        TOGGLE_TREE_PICKER_NODE_COLLAPSE: ({ id }) =>
            setNodeValueWith(toggleCollapse)(id)(state),
        TOGGLE_TREE_PICKER_NODE_SELECT: ({ id }) =>
            mapTreeValues(toggleSelect(id))(state),
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

const receiveNodes = (nodes: Array<TreePickerNode>) => (parent: string) => (state: TreePicker) =>
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