// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, CopyIcon, MoveToIcon } from "~/components/icon/icon";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveProjectDialog } from '~/store/projects/project-move-actions';
import { openProjectCreateDialog } from '~/store/projects/project-create-actions';
import { openProjectUpdateDialog } from '~/store/projects/project-update-actions';
import { ToggleTrashAction } from "~/views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "~/store/trash/trash-actions";

export const projectActionSet: ContextMenuActionSet = [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch<any>(openProjectCreateDialog(resource.uuid));
        }
    },
    {
        icon: RenameIcon,
        name: "Edit project",
        execute: (dispatch, resource) => {
            dispatch<any>(openProjectUpdateDialog(resource));
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
        component: ToggleTrashAction,
        execute: (dispatch, resource) => {
            dispatch<any>(toggleProjectTrashed(resource));
        }
    },
    {
        icon: MoveToIcon,
        name: "Move to",
        execute: (dispatch, resource) => dispatch<any>(openMoveProjectDialog(resource))
    },
    {
        icon: CopyIcon,
        name: "Copy to project",
        execute: (dispatch, resource) => {
            // add code
        }
    },
]];
