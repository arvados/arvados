// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../../store/store";
import { DownloadAction } from "./download-action";
import { getNodeValue } from "../../../models/tree";
import { ContextMenuKind } from '../context-menu';
import { filterCollectionFilesBySelection } from "store/collection-panel/collection-panel-files/collection-panel-files-state";
import { sanitizeToken } from "./helpers";

const mapStateToProps = (state: RootState) => {
    const { resource } = state.contextMenu;
    const currentCollectionUuid = state.collectionPanel.item ? state.collectionPanel.item.uuid : '';
    if (resource && [
        ContextMenuKind.COLLECTION_FILE_ITEM,
        ContextMenuKind.READONLY_COLLECTION_FILE_ITEM,
        ContextMenuKind.COLLECTION_DIRECTORY_ITEM,
        ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM ].indexOf(resource.menuKind as ContextMenuKind) > -1) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
            return {
                href: sanitizeToken(file.url, true),
                kind: 'file',
                currentCollectionUuid
            };
        }
    } else {
        const files = filterCollectionFilesBySelection(state.collectionPanelFiles, true);
        return {
            href: files.map(file => file.url),
            kind: 'files',
            currentCollectionUuid
        };
    }
    return {};
};

export const DownloadCollectionFileAction = connect(mapStateToProps)(DownloadAction);
