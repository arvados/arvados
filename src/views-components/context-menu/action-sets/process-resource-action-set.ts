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

export const readOnlyProcessResourceActionSet: ContextMenuActionSet = [[
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
            dispatch<any>(toggleDetailsPanel());
        }
    },
]];

export const processResourceActionSet: ContextMenuActionSet = [[
    ...readOnlyProcessResourceActionSet.reduce((prev, next) => prev.concat(next), []),
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
        name: "Remove",
        icon: RemoveIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openRemoveProcessDialog(resource.uuid));
        }
    }
]];
