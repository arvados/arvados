// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { CollectionPanelFiles as Component, CollectionPanelFilesProps } from "~/components/collection-panel-files/collection-panel-files";
import { RootState } from "~/store/store";
import { TreeItemStatus, TreeItem } from "~/components/tree/tree";
import { CollectionPanelFilesState, CollectionPanelDirectory, CollectionPanelFile } from "~/store/collection-panel/collection-panel-files/collection-panel-files-state";
import { FileTreeData } from "~/components/file-tree/file-tree-data";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "~/store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { contextMenuActions } from "~/store/context-menu/context-menu-actions";
import { ContextMenuKind } from "../context-menu/context-menu";
import { Tree, getNodeChildrenIds, getNode } from "~/models/tree";
import { CollectionFileType } from "~/models/collection-file";
import { openUploadCollectionFilesDialog } from '~/store/collections/uploader/collection-uploader-actions';

const memoizedMapStateToProps = () => {
    let prevState: CollectionPanelFilesState;
    let prevTree: Array<TreeItem<FileTreeData>>;

    return (state: RootState): Pick<CollectionPanelFilesProps, "items"> => {
        if (prevState !== state.collectionPanelFiles) {
            prevState = state.collectionPanelFiles;
            prevTree = getNodeChildrenIds('')(state.collectionPanelFiles)
                .map(collectionItemToTreeItem(state.collectionPanelFiles));
        }
        return {
            items: prevTree
        };
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onUploadDataClick' | 'onCollapseToggle' | 'onSelectionToggle' | 'onItemMenuOpen' | 'onOptionsMenuOpen'> => ({
    onUploadDataClick: () => {
        dispatch<any>(openUploadCollectionFilesDialog());
    },
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


export const CollectionPanelFiles = connect(memoizedMapStateToProps(), mapDispatchToProps)(Component);

const collectionItemToTreeItem = (tree: Tree<CollectionPanelDirectory | CollectionPanelFile>) =>
    (id: string): TreeItem<FileTreeData> => {
        const node = getNode(id)(tree) || {
            id: '',
            children: [],
            parent: '',
            value: {
                name: 'Invalid node',
                type: CollectionFileType.DIRECTORY,
                selected: false,
                collapsed: true
            }
        };
        return {
            active: false,
            data: {
                name: node.value.name,
                size: node.value.type === CollectionFileType.FILE ? node.value.size : undefined,
                type: node.value.type
            },
            id: node.id,
            items: getNodeChildrenIds(node.id)(tree)
                .map(collectionItemToTreeItem(tree)),
            open: node.value.type === CollectionFileType.DIRECTORY ? !node.value.collapsed : false,
            selected: node.value.selected,
            status: TreeItemStatus.LOADED
        };
    };
