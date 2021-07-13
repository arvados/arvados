// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../../store/store";
import { FileViewerAction } from 'views-components/context-menu/actions/file-viewer-action';
import { getNodeValue } from "models/tree";
import { ContextMenuKind } from 'views-components/context-menu/context-menu';
import { getInlineFileUrl, sanitizeToken } from "./helpers";

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
            const fileUrl = sanitizeToken(getInlineFileUrl(
                file.url,
                state.auth.config.keepWebServiceUrl,
                state.auth.config.keepWebInlineServiceUrl), true);
            return {
                href: fileUrl,
                kind: 'file',
                currentCollectionUuid
            };
        }
    }
    return {};
};

export const CollectionFileViewerAction = connect(mapStateToProps)(FileViewerAction);
