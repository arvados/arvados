// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { CollectionPanelFiles as Component, CollectionPanelFilesProps } from "../../components/collection-panel-files/collection-panel-files";
import { RootState } from "../../store/store";
import { TreeItemStatus, TreeItem } from "../../components/tree/tree";
import { CollectionPanelItem } from "../../store/collection-panel/collection-panel-files/collection-panel-files-state";
import { FileTreeData } from "../../components/file-tree/file-tree-data";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "../../store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { contextMenuActions } from "../../store/context-menu/context-menu-actions";
import { ContextMenuKind } from "../context-menu/context-menu";

const mapStateToProps = (state: RootState): Pick<CollectionPanelFilesProps, "items"> => ({
    items: state.collectionPanelFiles
        .filter(item => item.parentId === '')
        .map(collectionItemToTreeItem(state.collectionPanelFiles))
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onCollapseToggle' | 'onSelectionToggle' | 'onItemMenuOpen' | 'onOptionsMenuOpen'> => ({
    onCollapseToggle: (id) => {
        dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_COLLAPSE({ id }));
    },
    onSelectionToggle: (event, item) => {
        dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: item.id }));
    },
    onItemMenuOpen: (event, item) => {
        event.preventDefault();
        dispatch(contextMenuActions.OPEN_CONTEXT_MENU({
            position: { x: event.clientX, y: event.clientY },
            resource: { kind: ContextMenuKind.COLLECTION_FILES_ITEM, name: item.data.name, uuid: item.id }
        }));
    },
    onOptionsMenuOpen: (event) =>
        dispatch(contextMenuActions.OPEN_CONTEXT_MENU({
            position: { x: event.clientX, y: event.clientY },
            resource: { kind: ContextMenuKind.COLLECTION_FILES, name: '', uuid: '' }
        }))
});


export const CollectionPanelFiles = connect(mapStateToProps, mapDispatchToProps)(Component);

const collectionItemToTreeItem = (items: CollectionPanelItem[]) => (item: CollectionPanelItem): TreeItem<FileTreeData> => {
    return {
        active: false,
        data: {
            name: item.name,
            size: item.type === 'file' ? item.size : undefined,
            type: item.type
        },
        id: item.id,
        items: items
            .filter(i => i.parentId === item.id)
            .map(collectionItemToTreeItem(items)),
        open: item.type === 'directory' ? !item.collapsed : false,
        selected: item.selected,
        status: TreeItemStatus.LOADED
    };
};
