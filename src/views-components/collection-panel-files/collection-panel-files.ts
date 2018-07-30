// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { CollectionPanelFiles as Component, CollectionPanelFilesProps } from "../../components/collection-panel-files/collection-panel-files";
import { RootState } from "../../store/store";
import { TreeItemStatus, TreeItem } from "../../components/tree/tree";
import { CollectionPanelFile } from "../../store/collection-panel/collection-panel-files/collection-panel-files-state";
import { FileTreeData } from "../../components/file-tree/file-tree-data";
import { Dispatch } from "redux";
import { collectionPanelFilesAction } from "../../store/collection-panel/collection-panel-files/collection-panel-files-actions";

const mapStateToProps = (state: RootState): Pick<CollectionPanelFilesProps, "items"> => ({
    items: state.collectionPanelFiles
        .filter(f => f.parentId === undefined)
        .map(fileToTreeItem(state.collectionPanelFiles))
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<CollectionPanelFilesProps, 'onCollapseToggle' | 'onSelectionToggle'> => ({
    onCollapseToggle: (id) => dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_COLLAPSE({ id })),
    onSelectionToggle: (event, item) => dispatch(collectionPanelFilesAction.TOGGLE_COLLECTION_FILE_SELECTION({id: item.id})),
});


export const CollectionPanelFiles = connect(mapStateToProps, mapDispatchToProps)(Component);

const fileToTreeItem = (files: CollectionPanelFile[]) => (file: CollectionPanelFile): TreeItem<FileTreeData> => {
    return {
        active: false,
        data: {
            name: file.name,
            size: file.size,
            type: file.type
        },
        id: file.id,
        items: files
            .filter(f => f.parentId === file.id)
            .map(fileToTreeItem(files)),
        open: !file.collapsed,
        selected: file.selected,
        status: TreeItemStatus.LOADED
    };
};
