// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import {
    CollectionPanelFiles as Component,
    CollectionPanelFilesProps
} from "components/collection-panel-files/collection-panel-files";
import { RootState } from "store/store";
import { TreeItemStatus } from "components/tree/tree";
import { VirtualTreeItem as TreeItem } from "components/tree/virtual-tree";
import {
    CollectionPanelDirectory,
    CollectionPanelFile,
    CollectionPanelFilesState
} from "store/collection-panel/collection-panel-files/collection-panel-files-state";
import { FileTreeData } from "components/file-tree/file-tree-data";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { ContextMenuKind } from "../context-menu/context-menu";
import { getNode, getNodeChildrenIds, Tree, TreeNode, initTreeNode } from "models/tree";
import { CollectionFileType, createCollectionDirectory } from "models/collection-file";
import { openContextMenu, openCollectionFilesContextMenu } from 'store/context-menu/context-menu-actions';
import { openUploadCollectionFilesDialog } from 'store/collections/collection-upload-actions';
import { ResourceKind } from "models/resource";
import { openDetailsPanel } from 'store/details-panel/details-panel-action';

const memoizedMapStateToProps = () => {
    let prevState: CollectionPanelFilesState;
    let prevTree: Array<TreeItem<FileTreeData>>;

    return (state: RootState): Pick<CollectionPanelFilesProps, "items" | "currentItemUuid"> => {
        if (prevState !== state.collectionPanelFiles) {
            prevState = state.collectionPanelFiles;
            prevTree = [].concat.apply(
                [], getNodeChildrenIds('')(state.collectionPanelFiles)
                    .map(collectionItemToList(0)(state.collectionPanelFiles)));
        }
        return {
            items: prevTree,
            currentItemUuid: state.detailsPanel.resourceUuid
        };
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onSearchChange' | 'onFileClick' | 'onUploadDataClick' | 'onCollapseToggle' | 'onSelectionToggle' | 'onItemMenuOpen' | 'onOptionsMenuOpen'> => ({
    onUploadDataClick: () => {
        dispatch<any>(openUploadCollectionFilesDialog());
    },
    onCollapseToggle: (id) => {
        dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_COLLAPSE({ id }));
    },
    onSelectionToggle: (event, item) => {
        dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({ id: item.id }));
    },
    onItemMenuOpen: (event, item, isWritable) => {
        const isDirectory = item.data.type === 'directory';
        dispatch<any>(openContextMenu(
            event,
            {
                menuKind: isWritable
                    ? isDirectory
                        ? ContextMenuKind.COLLECTION_DIRECTORY_ITEM
                        : ContextMenuKind.COLLECTION_FILE_ITEM
                    : isDirectory
                        ? ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM
                        : ContextMenuKind.READONLY_COLLECTION_FILE_ITEM,
                kind: ResourceKind.COLLECTION,
                name: item.data.name,
                uuid: item.id,
                ownerUuid: ''
            }
        ));
    },
    onSearchChange: (searchValue: string) => {
        dispatch(collectionPanelFilesAction.ON_SEARCH_CHANGE(searchValue));
    },
    onOptionsMenuOpen: (event, isWritable) => {
        dispatch<any>(openCollectionFilesContextMenu(event, isWritable));
    },
    onFileClick: (id) => {
        dispatch<any>(openDetailsPanel(id));
    },
});

export const CollectionPanelFiles = connect(memoizedMapStateToProps(), mapDispatchToProps)(Component);

const collectionItemToList = (level: number) => (tree: Tree<CollectionPanelDirectory | CollectionPanelFile>) =>
    (id: string): TreeItem<FileTreeData>[] => {
        const node: TreeNode<CollectionPanelDirectory | CollectionPanelFile> = getNode(id)(tree) || initTreeNode({
            id: '',
            parent: '',
            value: {
                ...createCollectionDirectory({ name: 'Invalid file' }),
                selected: false,
                collapsed: true
            }
        });

        const treeItem = {
            active: false,
            data: {
                name: node.value.name,
                size: node.value.type === CollectionFileType.FILE ? node.value.size : undefined,
                type: node.value.type,
                url: node.value.url,
            },
            id: node.id,
            items: [], // Not used in this case as we're converting a tree to a list.
            itemCount: node.children.length,
            open: node.value.type === CollectionFileType.DIRECTORY ? !node.value.collapsed : false,
            selected: node.value.selected,
            status: TreeItemStatus.LOADED,
            level,
        };

        const treeItemChilds = treeItem.open
            ? [].concat.apply([], node.children.map(collectionItemToList(level+1)(tree)))
            : [];

        return [
            treeItem,
            ...treeItemChilds,
        ];
    };
