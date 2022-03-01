// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import {
    RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon,
    RemoveIcon, ReRunProcessIcon, InputIcon, OutputIcon, CommandIcon,
    LogIcon, AdvancedIcon
} from "components/icon/icon";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveProcessDialog } from 'store/processes/process-move-actions';
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { openCopyProcessDialog } from 'store/processes/process-copy-actions';
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openRemoveProcessDialog, reRunProcess } from "store/processes/processes-actions";
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { openProcessInputDialog } from "store/processes/process-input-actions";
import { navigateToOutput } from "store/process-panel/process-panel-actions";
import { openProcessCommandDialog } from "store/processes/process-command-actions";
import { navigateToProcessLogs } from "store/navigation/navigation-action";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { TogglePublicFavoriteAction } from "../actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

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
        icon: ReRunProcessIcon,
        name: "Re-run process",
        execute: (dispatch, resource) => {
            if(resource.workflowUuid) {
                dispatch<any>(reRunProcess(resource.uuid, resource.workflowUuid));
            } else {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: `You can't re-run this process`, hideDuration: 2000, kind: SnackbarKind.ERROR }));
            }
        }
    },
    {
        icon: InputIcon,
        name: "Inputs",
        execute: (dispatch, resource) => {
            dispatch<any>(openProcessInputDialog(resource.uuid));
        }
    },
    {
        icon: OutputIcon,
        name: "Outputs",
        execute: (dispatch, resource) => {
            if(resource.outputUuid){
                dispatch<any>(navigateToOutput(resource.outputUuid));
            }
        }
    },
    {
        icon: CommandIcon,
        name: "Command",
        execute: (dispatch, resource) => {
            dispatch<any>(openProcessCommandDialog(resource.uuid));
        }
    },
    {
        icon: LogIcon,
        name: "Log",
        execute: (dispatch, resource) => {
            dispatch<any>(navigateToProcessLogs(resource.uuid));
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

export const processResourceAdminActionSet: ContextMenuActionSet = [[
    ...processResourceActionSet.reduce((prev, next) => prev.concat(next), []),
    {
        component: TogglePublicFavoriteAction,
        name: "Add to public favorites",
        execute: (dispatch, resource) => {
            dispatch<any>(togglePublicFavorite(resource)).then(() => {
                dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
]];