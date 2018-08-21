// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { RenameIcon, ShareIcon, MoveToIcon, CopyIcon, DetailsIcon, ProvenanceGraphIcon, AdvancedIcon, RemoveIcon } from "~/components/icon/icon";
import { openUpdater } from "~/store/collections/updater/collection-updater-action";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveCollectionDialog } from '~/store/move-collection-dialog/move-collection-dialog';
import { openProjectCopyDialog } from "~/views-components/project-copy-dialog/project-copy-dialog";

export const collectionActionSet: ContextMenuActionSet = [[
    {
        icon: RenameIcon,
        name: "Edit collection",
        execute: (dispatch, resource) => {
            dispatch<any>(openUpdater(resource));
        }
    },
    {
        icon: ShareIcon,
        name: "Share",
        execute: (dispatch, resource) => {
            // add code
        }
    },
    {
        icon: MoveToIcon,
        name: "Move to",
        execute: (dispatch, resource) => dispatch<any>(openMoveCollectionDialog(resource))
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
            dispatch<any>(openProjectCopyDialog({name: resource.name, projectUuid: resource.uuid}));
        }
    },
    {
        icon: DetailsIcon,
        name: "View details",
        execute: (dispatch, resource) => {
            // add code
        }
    },
    {
        icon: ProvenanceGraphIcon,
        name: "Provenance graph",
        execute: (dispatch, resource) => {
            // add code
        }
    },
    {
        icon: AdvancedIcon,
        name: "Advanced",
        execute: (dispatch, resource) => {
            // add code
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
