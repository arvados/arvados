// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon, RemoveIcon } from "~/components/icon/icon";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveProcessDialog } from '~/store/processes/process-move-actions';
import { openProcessUpdateDialog } from "~/store/processes/process-update-actions";
import { openCopyProcessDialog } from '~/store/processes/process-copy-actions';
import { detailsPanelActions } from '~/store/details-panel/details-panel-action';
import { openSharingDialog } from "~/store/sharing-dialog/sharing-dialog-actions";

export const processResourceActionSet: ContextMenuActionSet = [[
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
        icon: MoveToIcon,
        name: "Move to",
        execute: (dispatch, resource) => {
            dispatch<any>(openMoveProcessDialog(resource));
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
            dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
        }
    },
    {
        icon: RemoveIcon,
        name: "Remove",
        execute: (dispatch, resource) => {
            // add code
        }
    }
]];
