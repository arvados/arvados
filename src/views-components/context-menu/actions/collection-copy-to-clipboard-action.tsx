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
    if (resource && resource.menuKind === ContextMenuKind.COLLECTION_FILES_ITEM) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
            return {
                href: file.url.replace(keepWebServiceUrl, ''),
                kind: 'file',
                currentCollectionUuid
            };
        }
    } else {
        return ;
    }
    return ;
};

export const CollectionCopyToClipboardAction = connect(mapStateToProps)(CopyToClipboardAction);
