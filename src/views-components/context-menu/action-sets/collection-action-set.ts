// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon, AdvancedIcon } from "~/components/icon/icon";
import { openCollectionUpdateDialog } from "~/store/collections/collection-update-actions";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveCollectionDialog } from '~/store/collections/collection-move-actions';
import { openCollectionCopyDialog } from "~/store/collections/collection-copy-actions";
import { ToggleTrashAction } from "~/views-components/context-menu/actions/trash-action";
import { toggleCollectionTrashed } from "~/store/trash/trash-actions";
import { openSharingDialog } from '~/store/sharing-dialog/sharing-dialog-actions';
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';

export const collectionActionSet: ContextMenuActionSet = [[
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
        execute: (dispatch, resource) => dispatch<any>(openMoveCollectionDialog(resource))
    },
    {
        icon: CopyIcon,
        name: "Copy to project",
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
        component: ToggleTrashAction,
        execute: (dispatch, resource) => {
            dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
        }
    },
    // {
    //     icon: RemoveIcon,
    //     name: "Remove",
    //     execute: (dispatch, resource) => {
    //         // add code
    //     }
    // }
]];
