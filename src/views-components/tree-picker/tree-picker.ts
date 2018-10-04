// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Tree, TreeProps, TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { RootState } from "~/store/store";
import { getNodeValue, getNodeChildrenIds, Tree as Ttree, createTree, getNode, TreeNodeStatus } from '~/models/tree';
import { Dispatch } from "redux";
import { initTreeNode } from '../../models/tree';

export interface TreePickerProps {
    pickerId: string;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, nodeId: string, pickerId: string) => void;
    toggleItemOpen: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
    toggleItemActive: (nodeId: string, status: TreeItemStatus, pickerId: string) => void;
    toggleItemSelection: (nodeId: string, pickerId: string) => void;
}

const memoizedMapStateToProps = () => {
    let prevTree: Ttree<any>;
    let mappedProps: Pick<TreeProps<any>, 'items'>;
    return (state: RootState, props: TreePickerProps): Pick<TreeProps<any>, 'items'> => {
        const tree = state.treePicker[props.pickerId] || createTree();
        if (tree !== prevTree) {
            prevTree = tree;
            mappedProps = {
                items: getNodeChildrenIds('')(tree)
                    .map(treePickerToTreeItems(tree))
            };
        }
        return mappedProps;
    };
};

const mapDispatchToProps = (dispatch: Dispatch, props: TreePickerProps): Pick<TreeProps<any>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive' | 'toggleItemSelection'> => ({
    onContextMenu: (event, item) => props.onContextMenu(event, item.id, props.pickerId),
    toggleItemActive: (id, status) => props.toggleItemActive(id, status, props.pickerId),
    toggleItemOpen: (id, status) => props.toggleItemOpen(id, status, props.pickerId),
    toggleItemSelection: (_, item) => props.toggleItemSelection(item.id, props.pickerId),
});

export const TreePicker = connect(memoizedMapStateToProps(), mapDispatchToProps)(Tree);

const treePickerToTreeItems = (tree: Ttree<any>) =>
    (id: string): TreeItem<any> => {
        const node = getNode(id)(tree) || initTreeNode({ id: '', value: 'InvalidNode' });
        const items = getNodeChildrenIds(node.id)(tree)
            .map(treePickerToTreeItems(tree));
        return {
            active: node.active,
            data: node.value,
            id: node.id,
            items: items.length > 0 ? items : undefined,
            open: node.expanded,
            selected: node.selected,
            status: treeNodeStatusToTreeItem(node.status),
        };
    };

export const treeNodeStatusToTreeItem = (status: TreeNodeStatus) => {
    switch (status) {
        case TreeNodeStatus.INITIAL:
            return TreeItemStatus.INITIAL;
        case TreeNodeStatus.PENDING:
            return TreeItemStatus.PENDING;
        case TreeNodeStatus.LOADED:
            return TreeItemStatus.LOADED;
    }
};

