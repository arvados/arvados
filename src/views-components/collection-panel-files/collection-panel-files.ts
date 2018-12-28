// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import {
    CollectionPanelFiles as Component,
    CollectionPanelFilesProps
} from "~/components/collection-panel-files/collection-panel-files";
import { RootState } from "~/store/store";
import { TreeItem, TreeItemStatus } from "~/components/tree/tree";
import {
    CollectionPanelDirectory,
    CollectionPanelFile,
    CollectionPanelFilesState
} from "~/store/collection-panel/collection-panel-files/collection-panel-files-state";
import { FileTreeData } from "~/components/file-tree/file-tree-data";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "~/store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { ContextMenuKind } from "../context-menu/context-menu";
import { getNode, getNodeChildrenIds, Tree, TreeNode, initTreeNode } from "~/models/tree";
import { CollectionFileType, createCollectionDirectory } from "~/models/collection-file";
import { openContextMenu, openCollectionFilesContextMenu } from '~/store/context-menu/context-menu-actions';
import { openUploadCollectionFilesDialog } from '~/store/collections/collection-upload-actions';
import { ResourceKind } from "~/models/resource";
import { openDetailsPanel } from '~/store/details-panel/details-panel-action';

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

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onFileClick' | 'onUploadDataClick' | 'onCollapseToggle' | 'onSelectionToggle' | 'onItemMenuOpen' | 'onOptionsMenuOpen'> => ({
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
        dispatch<any>(openContextMenu(event, { menuKind: ContextMenuKind.COLLECTION_FILES_ITEM, kind: ResourceKind.COLLECTION, name: item.data.name, uuid: item.id, ownerUuid: '' }));
    },
    onOptionsMenuOpen: (event) => {
        dispatch<any>(openCollectionFilesContextMenu(event));
    },
    onFileClick: (id) => {
        dispatch(openDetailsPanel(id));
    },
});


export const CollectionPanelFiles = connect(memoizedMapStateToProps(), mapDispatchToProps)(Component);

const collectionItemToTreeItem = (tree: Tree<CollectionPanelDirectory | CollectionPanelFile>) =>
    (id: string): TreeItem<FileTreeData> => {
        const node: TreeNode<CollectionPanelDirectory | CollectionPanelFile> = getNode(id)(tree) || initTreeNode({
            id: '',
            parent: '',
            value: {
                ...createCollectionDirectory({ name: 'Invalid file' }),
                selected: false,
                collapsed: true
            }
        });
        return {
            active: false,
            data: {
                name: node.value.name,
                size: node.value.type === CollectionFileType.FILE ? node.value.size : undefined,
                type: node.value.type,
                url: node.value.url,
            },
            id: node.id,
            items: getNodeChildrenIds(node.id)(tree)
                .map(collectionItemToTreeItem(tree)),
            open: node.value.type === CollectionFileType.DIRECTORY ? !node.value.collapsed : false,
            selected: node.value.selected,
            status: TreeItemStatus.LOADED
        };
    };
