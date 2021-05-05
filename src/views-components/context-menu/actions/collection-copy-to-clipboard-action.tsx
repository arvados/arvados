// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../../store/store";
import { getNodeValue } from "~/models/tree";
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { CopyToClipboardAction } from "./copy-to-clipboard-action";

const mapStateToProps = (state: RootState) => {
    const { resource } = state.contextMenu;
    const currentCollectionUuid = state.collectionPanel.item ? state.collectionPanel.item.uuid : '';
    const { keepWebServiceUrl } = state.auth.config;
    if (resource && [
        ContextMenuKind.COLLECTION_FILE_ITEM,
        ContextMenuKind.READONLY_COLLECTION_FILE_ITEM,
        ContextMenuKind.COLLECTION_DIRECTORY_ITEM,
        ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM ].indexOf(resource.menuKind as ContextMenuKind) > -1) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
            return {
                href: file.url.replace(keepWebServiceUrl, ''),
                kind: 'file',
                currentCollectionUuid
            };
        }
    }
    return {};
};

export const CollectionCopyToClipboardAction = connect(mapStateToProps)(CopyToClipboardAction);
