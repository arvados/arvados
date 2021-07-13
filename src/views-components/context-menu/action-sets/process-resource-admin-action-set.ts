// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import { RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon, RemoveIcon } from "components/icon/icon";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveProcessDialog } from 'store/processes/process-move-actions';
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { openCopyProcessDialog } from 'store/processes/process-copy-actions';
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { TogglePublicFavoriteAction } from "views-components/context-menu/actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

export const processResourceAdminActionSet: ContextMenuActionSet = [[
    {
        icon: RenameIcon,
        name: "Edit process",
        execute: (dispatch, resource) => {
            dispatch<any>(openProcessUpdateDialog(resource));
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
        component: ToggleFavoriteAction,
        execute: (dispatch, resource) => {
            dispatch<any>(toggleFavorite(resource)).then(() => {
                dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
    {
        component: TogglePublicFavoriteAction,
        execute: (dispatch, resource) => {
            dispatch<any>(togglePublicFavorite(resource)).then(() => {
                dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
    {
        icon: MoveToIcon,
        name: "Move to",
        execute: (dispatch, resource) => {
            dispatch<any>(openMoveProcessDialog(resource));
        }
    },
    {
        icon: CopyIcon,
        name: "Copy to project",
        execute: (dispatch, resource) => {
            dispatch<any>(openCopyProcessDialog(resource));
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
        name: "Remove",
        icon: RemoveIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openRemoveProcessDialog(resource.uuid));
        }
    }
]];
