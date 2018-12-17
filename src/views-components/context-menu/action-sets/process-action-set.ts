// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { ToggleFavoriteAction } from "~/views-components/context-menu/actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import {
    RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon, ProvenanceGraphIcon,
    AdvancedIcon, RemoveIcon, ReRunProcessIcon, LogIcon, InputIcon, CommandIcon, OutputIcon
} from "~/components/icon/icon";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { navigateToProcessLogs } from '~/store/navigation/navigation-action';
import { openMoveProcessDialog } from '~/store/processes/process-move-actions';
import { openProcessUpdateDialog } from "~/store/processes/process-update-actions";
import { openCopyProcessDialog } from '~/store/processes/process-copy-actions';
import { openProcessCommandDialog } from '~/store/processes/process-command-actions';
import { openSharingDialog } from "~/store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { openProcessInputDialog } from "~/store/processes/process-input-actions";
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';
import { openRemoveProcessDialog } from "~/store/processes/processes-actions";
import { navigateToOutput } from "~/store/process-panel/process-panel-actions";

export const processActionSet: ContextMenuActionSet = [[
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
        icon: ReRunProcessIcon,
        name: "Re-run process",
        execute: (dispatch, resource) => {
            // add code
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
    // {
    //     icon: ProvenanceGraphIcon,
    //     name: "Provenance graph",
    //     execute: (dispatch, resource) => {
    //         // add code
    //     }
    // },
    {
        icon: AdvancedIcon,
        name: "Advanced",
        execute: (dispatch, resource) => {
            dispatch<any>(openAdvancedTabDialog(resource.uuid));
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
