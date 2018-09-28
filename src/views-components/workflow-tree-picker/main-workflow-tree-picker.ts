// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Tree, TreeProps, TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { RootState } from "~/store/store";
import { createTreePickerNode, TreePickerNode } from "~/store/workflow-tree-picker/workflow-tree-picker";
import { getNodeValue, getNodeChildrenIds, Tree as Ttree, createTree } from "~/models/tree";
import { Dispatch } from "redux";

export interface MainWorkflowTreePickerProps {
    pickerId: string;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, nodeId: string, pickerId: string) => void;
    toggleItemOpen: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
    toggleItemActive: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
}

const memoizedMapStateToProps = () => {
    let prevTree: Ttree<TreePickerNode>;
    let mappedProps: Pick<TreeProps<any>, 'items'>;
    return (state: RootState, props: MainWorkflowTreePickerProps): Pick<TreeProps<any>, 'items'> => {
        const tree = state.treePicker[props.pickerId] || createTree();
        if(tree !== prevTree){
            prevTree = tree;
            mappedProps = {
                items: getNodeChildrenIds('')(tree)
                    .map(treePickerToTreeItems(tree))
            };
        }
        return mappedProps;
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: MainWorkflowTreePickerProps): Pick<TreeProps<any>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive'> => ({
    onContextMenu: (event, item) => props.onContextMenu(event, item.id, props.pickerId),
    toggleItemActive: (id, status) => props.toggleItemActive(id, status, props.pickerId),
    toggleItemOpen: (id, status) => props.toggleItemOpen(id, status, props.pickerId)
});

export const MainWorkflowTreePicker = connect(memoizedMapStateToProps(), mapDispatchToProps)(Tree);

const treePickerToTreeItems = (tree: Ttree<TreePickerNode>) =>
    (id: string): TreeItem<any> => {
        const node: TreePickerNode = getNodeValue(id)(tree) || createTreePickerNode({ nodeId: '', value: 'InvalidNode' });
        const items = getNodeChildrenIds(node.nodeId)(tree)
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

