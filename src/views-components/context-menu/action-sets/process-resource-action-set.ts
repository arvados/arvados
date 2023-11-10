// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import {
    RenameIcon,
    ShareIcon,
    MoveToIcon,
    DetailsIcon,
    RemoveIcon,
    ReRunProcessIcon,
    OutputIcon,
    AdvancedIcon,
    OpenIcon,
    StopIcon,
} from "components/icon/icon";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveProcessDialog } from "store/processes/process-move-actions";
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { toggleDetailsPanel } from "store/details-panel/details-panel-action";
import { navigateToOutput } from "store/process-panel/process-panel-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { TogglePublicFavoriteAction } from "../actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { cancelRunningWorkflow } from "store/processes/processes-actions";

export const readOnlyProcessResourceActionSet: ContextMenuActionSet = [
    [
        {
            component: ToggleFavoriteAction,
            execute: (dispatch, resources) => {
                dispatch<any>(toggleFavorite(resources[0])).then(() => {
                    dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
                });
            },
        },
        {
            icon: OpenIcon,
            name: "Open in new tab",
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            icon: ReRunProcessIcon,
            name: "Copy and re-run process",
            execute: (dispatch, resources) => {
                dispatch<any>(openCopyProcessDialog(resources[0]));
            },
        },
        {
            icon: OutputIcon,
            name: "Outputs",
            execute: (dispatch, resources) => {
                if (resources[0].outputUuid) {
                    dispatch<any>(navigateToOutput(resources[0].outputUuid));
                }
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

export const processResourceActionSet: ContextMenuActionSet = [
    [
        ...readOnlyProcessResourceActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RenameIcon,
            name: "Edit process",
            execute: (dispatch, resources) => {
                dispatch<any>(openProcessUpdateDialog(resources[0]));
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
            execute: (dispatch, resources) => {
                dispatch<any>(openMoveProcessDialog(resources[0]));
            },
        },
        {
            name: "Remove",
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
            },
        },
    ],
];

const runningProcessOnlyActionSet: ContextMenuActionSet = [
    [
        {
            name: "CANCEL",
            icon: StopIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(cancelRunningWorkflow(resources[0].uuid));
            },
        },
    ]
];

export const processResourceAdminActionSet: ContextMenuActionSet = [
    [
        ...processResourceActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            component: TogglePublicFavoriteAction,
            name: "Add to public favorites",
            execute: (dispatch, resources) => {
                dispatch<any>(togglePublicFavorite(resources[0])).then(() => {
                    dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
                });
            },
        },
    ],
];

export const runningProcessResourceActionSet = [
    [
        ...processResourceActionSet.reduce((prev, next) => prev.concat(next), []),
        ...runningProcessOnlyActionSet.reduce((prev, next) => prev.concat(next), []),
    ],
];

export const runningProcessResourceAdminActionSet: ContextMenuActionSet = [
    [
        ...processResourceAdminActionSet.reduce((prev, next) => prev.concat(next), []),
        ...runningProcessOnlyActionSet.reduce((prev, next) => prev.concat(next), []),
    ],
];
