// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Tree, TreeProps, TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { RootState } from "~/store/store";
import { getNodeChildrenIds, Tree as Ttree, createTree, getNode, TreeNodeStatus } from '~/models/tree';
import { Dispatch } from "redux";
import { initTreeNode } from '../../models/tree';

type Callback<T> = (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>, pickerId: string) => void;
export interface TreePickerProps<T> {
    pickerId: string;
    onContextMenu: Callback<T>;
    toggleItemOpen: Callback<T>;
    toggleItemActive: Callback<T>;
    toggleItemSelection: Callback<T>;
}

const flatTree = (depth: number, items?: any): [] => {
    return items ? items.reduce((prev: any, next: any) => {
        const { items } = next;
        // delete next.items;
        return [
            ...prev,
            { ...next, depth },
            ...(next.open ? flatTree(depth + 1, items) : []),
        ];
    }, []) : [];
};

const memoizedMapStateToProps = () => {
    let prevTree: Ttree<any>;
    let mappedProps: Pick<TreeProps<any>, 'items' | 'disableRipple'>;
    return <T>(state: RootState, props: TreePickerProps<T>): Pick<TreeProps<T>, 'items' | 'disableRipple'> => {
        const tree = state.treePicker[props.pickerId] || createTree();
        if (tree !== prevTree) {
            prevTree = tree;
            mappedProps = {
                disableRipple: true,
                items: getNodeChildrenIds('')(tree)
                    .map(treePickerToTreeItems(tree))
                    .map(parentItem => ({
                        ...parentItem,
                        flatTree: true,
                        items: flatTree(2, parentItem.items || []),
                    }))
            };
        }
        return mappedProps;
    };
};

const mapDispatchToProps = (_: Dispatch, props: TreePickerProps<any>): Pick<TreeProps<any>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive' | 'toggleItemSelection'> => ({
    onContextMenu: (event, item) => props.onContextMenu(event, item, props.pickerId),
    toggleItemActive: (event, item) => props.toggleItemActive(event, item, props.pickerId),
    toggleItemOpen: (event, item) => props.toggleItemOpen(event, item, props.pickerId),
    toggleItemSelection: (event, item) => props.toggleItemSelection(event, item, props.pickerId),
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

