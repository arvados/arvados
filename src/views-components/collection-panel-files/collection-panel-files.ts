// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import {
    CollectionPanelFiles as Component,
    CollectionPanelFilesProps
} from "components/collection-panel-files/collection-panel-files";
import { RootState } from "store/store";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { ContextMenuKind } from "../context-menu/context-menu";
import { openContextMenu, openCollectionFilesContextMenu } from 'store/context-menu/context-menu-actions';
import { openUploadCollectionFilesDialog } from 'store/collections/collection-upload-actions';
import { ResourceKind } from "models/resource";
import { openDetailsPanel } from 'store/details-panel/details-panel-action';

const mapStateToProps = (state: RootState): Pick<CollectionPanelFilesProps, "currentItemUuid"> => ({
    currentItemUuid: state.detailsPanel.resourceUuid
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onSearchChange' | 'onFileClick' | 'onUploadDataClick' | 'onCollapseToggle' | 'onSelectionToggle' | 'onItemMenuOpen' | 'onOptionsMenuOpen'> => ({
    onUploadDataClick: (targetLocation?: string) => {
        dispatch<any>(openUploadCollectionFilesDialog(targetLocation));
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

export const CollectionPanelFiles = connect(mapStateToProps, mapDispatchToProps)(Component);
