// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction, ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import {
    RenameIcon,
    ShareIcon,
    MoveToIcon,
    CopyIcon,
    DetailsIcon,
    AdvancedIcon,
    OpenIcon,
    Link,
    RestoreVersionIcon,
    FolderSharedIcon,
} from "components/icon/icon";
import { openCollectionUpdateDialog } from "store/collections/collection-update-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveCollectionDialog } from "store/collections/collection-move-actions";
import { openCollectionCopyDialog, openMultiCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from "store/details-panel/details-panel-action";
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openRestoreCollectionVersionDialog } from "store/collections/collection-version-actions";
import { TogglePublicFavoriteAction } from "../actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

const toggleFavoriteAction: ContextMenuAction = {
    component: ToggleFavoriteAction,
    name: "ToggleFavoriteAction",
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch<any>(toggleFavorite(resource)).then(() => {
                dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
};
const commonActionSet: ContextMenuActionSet = [
    [
        {
            icon: OpenIcon,
            name: "Open in new tab",
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            icon: Link,
            name: "Copy to clipboard",
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            icon: CopyIcon,
            name: "Make a copy",
            execute: (dispatch, resources) => {
                if (resources[0].isSingle || resources.length === 1) dispatch<any>(openCollectionCopyDialog(resources[0]));
                else dispatch<any>(openMultiCollectionCopyDialog(resources[0]));
            },
        },
        {
            icon: DetailsIcon,
            name: "View details",
            execute: dispatch => {
                dispatch<any>(toggleDetailsPanel());
            },
        },
        {
            icon: AdvancedIcon,
            name: "API Details",
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
    ],
];

export const readOnlyCollectionActionSet: ContextMenuActionSet = [
    [
        ...commonActionSet.reduce((prev, next) => prev.concat(next), []),
        toggleFavoriteAction,
        {
            icon: FolderSharedIcon,
            name: "Open with 3rd party client",
            execute: (dispatch, resources) => {
                dispatch<any>(openWebDavS3InfoDialog(resources[0].uuid));
            },
        },
    ],
];

export const collectionActionSet: ContextMenuActionSet = [
    [
        ...readOnlyCollectionActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RenameIcon,
            name: "Edit collection",
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionUpdateDialog(resources[0]));
            },
        },
        {
            icon: ShareIcon,
            name: "Share",
            execute: (dispatch, resources) => {
                dispatch<any>(openSharingDialog(resources[0].uuid));
            },
        },
        {
            icon: MoveToIcon,
            name: "Move to",
            execute: (dispatch, resources) => dispatch<any>(openMoveCollectionDialog(resources[0])),
        },
        {
            component: ToggleTrashAction,
            name: "ToggleTrashAction",
            execute: (dispatch, resources: ContextMenuResource[]) => {
                for (const resource of resources) {
                    dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
                }
            },
        },
    ],
];

export const collectionAdminActionSet: ContextMenuActionSet = [
    [
        ...collectionActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            component: TogglePublicFavoriteAction,
            name: "TogglePublicFavoriteAction",
            execute: (dispatch, resources) => {
                for (const resource of resources) {
                    dispatch<any>(togglePublicFavorite(resource)).then(() => {
                        dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
                    });
                }
            },
        },
    ],
];

export const oldCollectionVersionActionSet: ContextMenuActionSet = [
    [
        ...commonActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RestoreVersionIcon,
            name: "Restore version",
            execute: (dispatch, resources) => {
                for (const resource of resources) {
                    dispatch<any>(openRestoreCollectionVersionDialog(resource.uuid));
                }
            },
        },
    ],
];
