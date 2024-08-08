// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../../store/store";
import { getNodeValue } from "models/tree";
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';
import { CopyToClipboardAction } from "./copy-to-clipboard-action";
import { replaceCollectionId, getCollectionItemClipboardUrl, sanitizeToken } from "./helpers";

const mapStateToPropsUUID = (state: RootState) => {
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
                href: getCollectionItemClipboardUrl(replaceCollectionId(file.url, currentCollectionUuid),
						    state.auth.config.keepWebServiceUrl,
						    state.auth.config.keepWebInlineServiceUrl),
                kind: 'file',
		customText: "Copy link to latest version (content may change if collection changes)",
	    };
        }
    }
    return {};
};

const mapStateToPropsPDH = (state: RootState) => {
    const { resource } = state.contextMenu;
    const currentCollectionPDH = state.collectionPanel.item ? state.collectionPanel.item.portableDataHash : '';
    if (resource && [
        ContextMenuKind.COLLECTION_FILE_ITEM,
        ContextMenuKind.READONLY_COLLECTION_FILE_ITEM,
        ContextMenuKind.COLLECTION_DIRECTORY_ITEM,
        ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM ].indexOf(resource.menuKind as ContextMenuKind) > -1) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
	    return {
                href: getCollectionItemClipboardUrl(replaceCollectionId(file.url, currentCollectionPDH),
						    state.auth.config.keepWebServiceUrl,
						    state.auth.config.keepWebInlineServiceUrl),
		kind: 'file',
		customText: "Copy link to this exact version (content is immutable)",
	    };
        }
    }
    return {};
};

const mapStateToPropsCWL = (state: RootState) => {
    const { resource } = state.contextMenu;
    const currentCollectionPDH = state.collectionPanel.item ? state.collectionPanel.item.portableDataHash : '';
    if (resource && [
        ContextMenuKind.COLLECTION_FILE_ITEM,
        ContextMenuKind.READONLY_COLLECTION_FILE_ITEM,
        ContextMenuKind.COLLECTION_DIRECTORY_ITEM,
        ContextMenuKind.READONLY_COLLECTION_DIRECTORY_ITEM ].indexOf(resource.menuKind as ContextMenuKind) > -1) {
        const file = getNodeValue(resource.uuid)(state.collectionPanelFiles);
        if (file) {
	    let url = file.url;
	    url = replaceCollectionId(url, '');
	    url = sanitizeToken(url, false);
	    const path = new URL(url).pathname;
	    return {
                href: `keep:${currentCollectionPDH}${path}`,
		kind: 'file',
		customText: "Copy file reference to use in CWL",
	    };
        }
    }
    return {};
};

export const CollectionUUIDCopyToClipboardAction = connect(mapStateToPropsUUID)(CopyToClipboardAction);

export const CollectionPDHCopyToClipboardAction = connect(mapStateToPropsPDH)(CopyToClipboardAction);

export const CollectionCWLCopyToClipboardAction = connect(mapStateToPropsCWL)(CopyToClipboardAction);
