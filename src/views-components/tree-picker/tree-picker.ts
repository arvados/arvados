// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Tree, TreeProps, TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { RootState } from "~/store/store";
import { createTreePickerNode, TreePickerNode } from "~/store/tree-picker/tree-picker";
import { getNodeValue, getNodeChildren, Tree as Ttree, createTree } from "~/models/tree";
import { Dispatch } from "redux";

export interface TreePickerProps {
    pickerId: string;
    toggleItemOpen: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
    toggleItemActive: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
}

const mapStateToProps = (state: RootState, props: TreePickerProps): Pick<TreeProps<any>, 'items'> => {
    const tree = state.treePicker[props.pickerId] || createTree();
    return {
        items: getNodeChildren('')(tree)
            .map(treePickerToTreeItems(tree))
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: TreePickerProps): Pick<TreeProps<any>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive'> => ({
    onContextMenu: () => { return; },
    toggleItemActive: (id, status) => props.toggleItemActive(id, status, props.pickerId),
    toggleItemOpen: (id, status) => props.toggleItemOpen(id, status, props.pickerId)
});

export const TreePicker = connect(mapStateToProps, mapDispatchToProps)(Tree);

const treePickerToTreeItems = (tree: Ttree<TreePickerNode>) =>
    (id: string): TreeItem<any> => {
        const node: TreePickerNode = getNodeValue(id)(tree) || createTreePickerNode({ nodeId: '', value: 'InvalidNode' });
        const items = getNodeChildren(node.nodeId)(tree)
            .map(treePickerToTreeItems(tree));
        return {
            active: node.selected,
            data: node.value,
            id: node.nodeId,
            items: items.length > 0 ? items : undefined,
            open: !node.collapsed,
            status: node.status
        };
    };

