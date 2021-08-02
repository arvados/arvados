// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    ContextMenuAction,
    ContextMenuActionSet
} from "../context-menu-action-set";
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
    FolderSharedIcon
} from "components/icon/icon";
import { openCollectionUpdateDialog } from "store/collections/collection-update-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveCollectionDialog } from 'store/collections/collection-move-actions';
import { openCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openRestoreCollectionVersionDialog } from "store/collections/collection-version-actions";
import { TogglePublicFavoriteAction } from "../actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

const toggleFavoriteAction: ContextMenuAction = {
    component: ToggleFavoriteAction,
    name: 'ToggleFavoriteAction',
    execute: (dispatch, resource) => {
        dispatch<any>(toggleFavorite(resource)).then(() => {
            dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
        });
    }
};

const commonActionSet: ContextMenuActionSet = [[
    {
        icon: OpenIcon,
        name: "Open in new tab",
        execute: (dispatch, resource) => {
            dispatch<any>(openInNewTabAction(resource));
        }
    },
    {
        icon: Link,
        name: "Copy to clipboard",
        execute: (dispatch, resource) => {
            dispatch<any>(copyToClipboardAction(resource));
        }
    },
    {
        icon: CopyIcon,
        name: "Make a copy",
        execute: (dispatch, resource) => {
            dispatch<any>(openCollectionCopyDialog(resource));
        }

    },
    {
        icon: DetailsIcon,
        name: "View details",
        execute: dispatch => {
            dispatch<any>(toggleDetailsPanel());
        }
    },
    {
        icon: AdvancedIcon,
        name: "Advanced",
        execute: (dispatch, resource) => {
            dispatch<any>(openAdvancedTabDialog(resource.uuid));
        }
    },
]];

export const readOnlyCollectionActionSet: ContextMenuActionSet = [[
    ...commonActionSet.reduce((prev, next) => prev.concat(next), []),
    toggleFavoriteAction,
    {
        icon: FolderSharedIcon,
        name: "Open with 3rd party client",
        execute: (dispatch, resource) => {
            dispatch<any>(openWebDavS3InfoDialog(resource.uuid));
        }
    },
]];

export const collectionActionSet: ContextMenuActionSet = [
    [
        ...readOnlyCollectionActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RenameIcon,
            name: "Edit collection",
            execute: (dispatch, resource) => {
                dispatch<any>(openCollectionUpdateDialog(resource));
            }
        },
        {
            icon: ShareIcon,
            name: "Share",
            execute: (dispatch, { uuid }) => {
                dispatch<any>(openSharingDialog(uuid));
            }
        },
        {
            icon: MoveToIcon,
            name: "Move to",
            execute: (dispatch, resource) => dispatch<any>(openMoveCollectionDialog(resource))
        },
        {
            component: ToggleTrashAction,
            name: 'ToggleTrashAction',
            execute: (dispatch, resource) => {
                dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
            }
        },
    ]
];

export const collectionAdminActionSet: ContextMenuActionSet = [
    [
        ...collectionActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            component: TogglePublicFavoriteAction,
            name: 'TogglePublicFavoriteAction',
            execute: (dispatch, resource) => {
                dispatch<any>(togglePublicFavorite(resource)).then(() => {
                    dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
                });
            }
        },
    ]
];

export const oldCollectionVersionActionSet: ContextMenuActionSet = [
    [
        ...commonActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RestoreVersionIcon,
            name: 'Restore version',
            execute: (dispatch, { uuid }) => {
                dispatch<any>(openRestoreCollectionVersionDialog(uuid));
            }
        },
    ]
];
