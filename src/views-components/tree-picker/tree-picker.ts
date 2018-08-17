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
    pickerKind: string;
    toggleItemOpen: (id: string, status: TreeItemStatus, pickerKind: string) => void;
    toggleItemActive: (id: string, status: TreeItemStatus, pickerKind: string) => void;
}

const mapStateToProps = (state: RootState, props: TreePickerProps): Pick<TreeProps<any>, 'items'> => {
    const tree = state.treePicker[props.pickerKind] || createTree();
    return {
        items: getNodeChildren('')(tree)
            .map(treePickerToTreeItems(tree))
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: TreePickerProps): Pick<TreeProps<any>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive'> => ({
    onContextMenu: () => { return; },
    toggleItemActive: (id, status) => props.toggleItemActive(id, status, props.pickerKind),
    toggleItemOpen: (id, status) => props.toggleItemOpen(id, status, props.pickerKind)
});

export const TreePicker = connect(mapStateToProps, mapDispatchToProps)(Tree);

const treePickerToTreeItems = (tree: Ttree<TreePickerNode>) =>
    (id: string): TreeItem<any> => {
        const node: TreePickerNode = getNodeValue(id)(tree) || createTreePickerNode({ id: '', value: 'InvalidNode' });
        const items = getNodeChildren(node.id)(tree)
            .map(treePickerToTreeItems(tree));
        return {
            active: node.selected,
            data: node.value,
            id: node.id,
            items: items.length > 0 ? items : undefined,
            open: !node.collapsed,
            status: node.status
        };
    };

