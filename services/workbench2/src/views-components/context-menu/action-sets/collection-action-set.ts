// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction, ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
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
    FileCopyOutlinedIcon,
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
import { openDetailsPanel } from "store/details-panel/details-panel-action";
import { copyToClipboardAction, copyStringToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openRestoreCollectionVersionDialog } from "store/collections/collection-version-actions";
import { TogglePublicFavoriteAction } from "../actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

const toggleFavoriteAction: ContextMenuAction = {
    component: ToggleFavoriteAction,
    name: ContextMenuActionNames.ADD_TO_FAVORITES,
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
            name: ContextMenuActionNames.OPEN_IN_NEW_TAB,
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            icon: Link,
            name: ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            icon: CopyIcon,
            name: ContextMenuActionNames.COPY_UUID,
            execute: (dispatch, resources) => {
                dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
            },
        },
        {
            icon: FileCopyOutlinedIcon,
            name: ContextMenuActionNames.MAKE_A_COPY,
            execute: (dispatch, resources) => {
                if (resources[0].fromContextMenu || resources.length === 1) dispatch<any>(openCollectionCopyDialog(resources[0]));
                else dispatch<any>(openMultiCollectionCopyDialog(resources[0]));
            },
        },
        {
            icon: DetailsIcon,
            name: ContextMenuActionNames.VIEW_DETAILS,
            execute: (dispatch, resources) => {
                dispatch<any>(openDetailsPanel(resources[0].uuid));
            },
        },
        {
            icon: AdvancedIcon,
            name: ContextMenuActionNames.API_DETAILS,
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
            name: ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
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
            name: ContextMenuActionNames.EDIT_COLLECTION,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionUpdateDialog(resources[0]));
            },
        },
        {
            icon: ShareIcon,
            name: ContextMenuActionNames.SHARE,
            execute: (dispatch, resources) => {
                dispatch<any>(openSharingDialog(resources[0].uuid));
            },
        },
        {
            icon: MoveToIcon,
            name: ContextMenuActionNames.MOVE_TO,
            execute: (dispatch, resources) => dispatch<any>(openMoveCollectionDialog(resources[0])),
        },
        {
            component: ToggleTrashAction,
            name: ContextMenuActionNames.MOVE_TO_TRASH,
            execute: (dispatch, resources: ContextMenuResource[]) => {
                for (const resource of [...resources]) {
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
            name: ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
            execute: (dispatch, resources) => {
                for (const resource of [...resources]) {
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
            name: ContextMenuActionNames.RESTORE_VERSION,
            execute: (dispatch, resources) => {
                for (const resource of [...resources]) {
                    dispatch<any>(openRestoreCollectionVersionDialog(resource.uuid));
                }
            },
        },
    ],
];

export const writeableCollectionSet: ContextMenuActionSet = [
    [
        ...collectionActionSet.reduce((prev, next) => {
            return prev.concat(next.filter(action => action.name !== ContextMenuActionNames.SHARE));
        }, []),
    ]
];
