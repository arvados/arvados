// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Tree, TreeProps, TreeItem } from "~/components/tree/tree";
import { RootState } from "~/store/store";
import { TreePicker as TTreePicker, TreePickerNode, createTreePickerNode } from "~/store/tree-picker/tree-picker";
import { getNodeValue, getNodeChildren } from "~/models/tree";

const memoizedMapStateToProps = () => {
    let prevState: TTreePicker;
    let prevTree: Array<TreeItem<any>>;

    return (state: RootState): Pick<TreeProps<any>, 'items'> => {
        if (prevState !== state.treePicker) {
            prevState = state.treePicker;
            prevTree = getNodeChildren('')(state.treePicker)
                .map(treePickerToTreeItems(state.treePicker));
        }
        return {
            items: prevTree
        };
    };
};

const mapDispatchToProps = (): Pick<TreeProps<any>, 'onContextMenu'> => ({
    onContextMenu: () => { return; },
});

export const TreePicker = connect(memoizedMapStateToProps(), mapDispatchToProps)(Tree);

const treePickerToTreeItems = (tree: TTreePicker) =>
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

