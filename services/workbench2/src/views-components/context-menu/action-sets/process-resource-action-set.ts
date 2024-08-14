// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import {
    RenameIcon,
    DetailsIcon,
    RemoveIcon,
    ReRunProcessIcon,
    OutputIcon,
    AdvancedIcon,
    OpenIcon,
    StopIcon,
} from "components/icon/icon";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { openDetailsPanel } from "store/details-panel/details-panel-action";
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
            name: ContextMenuActionNames.ADD_TO_FAVORITES,
            execute: (dispatch, resources) => {
                dispatch<any>(toggleFavorite(resources[0])).then(() => {
                    dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
                });
            },
        },
        {
            icon: OpenIcon,
            name: ContextMenuActionNames.OPEN_IN_NEW_TAB,
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            icon: ReRunProcessIcon,
            name: ContextMenuActionNames.COPY_AND_RERUN_PROCESS,
            execute: (dispatch, resources) => {
                dispatch<any>(openCopyProcessDialog(resources[0]));
            },
        },
        {
            icon: OutputIcon,
            name: ContextMenuActionNames.OUTPUTS,
            execute: (dispatch, resources) => {
                if (resources[0]) {
                    dispatch<any>(navigateToOutput(resources[0]));
                }
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

export const processResourceActionSet: ContextMenuActionSet = [
    [
        ...readOnlyProcessResourceActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RenameIcon,
            name: ContextMenuActionNames.EDIT_PROCESS,
            execute: (dispatch, resources) => {
                dispatch<any>(openProcessUpdateDialog(resources[0]));
            },
        },
        // removed until auto-move children is implemented
        // {
        //     icon: MoveToIcon,
        //     name: ContextMenuActionNames.MOVE_TO,
        //     execute: (dispatch, resources) => {
        //         dispatch<any>(openMoveProcessDialog(resources[0]));
        //     },
        // },
        {
            name: ContextMenuActionNames.REMOVE,
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
            name: ContextMenuActionNames.CANCEL,
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
            name: ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
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
