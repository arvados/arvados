// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { TreeComponent, TreeProps, TreeItem} from "components/tree/tree";
import { RootState } from "store/store";
import { Tree, createTree} from 'models/tree';
import { Dispatch } from "redux";

type Callback<T> = (event: React.MouseEvent<HTMLElement>, item: TreeItem<T>, pickerId: string) => void;
export interface TreePickerProps<T> {
    pickerId: string;
    onContextMenu: Callback<T>;
    toggleItemOpen: Callback<T>;
    toggleItemActive: Callback<T>;
    toggleItemSelection: Callback<T>;
}

const mapStateToProps =
    <T>(state: RootState, props: TreePickerProps<T>): Pick<TreeProps<T>, 'tree' | 'resources'> => {
        const tree: Tree<T> = state.treePicker[props.pickerId] || createTree<T>();
        return {
            tree: tree,
            resources: state.resources,
        };
    };

const mapDispatchToProps = <T>(_: Dispatch, props: TreePickerProps<T>): Pick<TreeProps<T>, 'onContextMenu' | 'toggleItemOpen' | 'toggleItemActive' | 'toggleItemSelection'> => ({
    onContextMenu: (event, item) => props.onContextMenu(event, item, props.pickerId),
    toggleItemActive: (event, item) => props.toggleItemActive(event, item, props.pickerId),
    toggleItemOpen: (event, item) => props.toggleItemOpen(event, item, props.pickerId),
    toggleItemSelection: (event, item) => props.toggleItemSelection(event, item, props.pickerId),
});

export const TreePicker = connect(mapStateToProps, mapDispatchToProps)(TreeComponent);

