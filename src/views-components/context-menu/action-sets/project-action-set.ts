// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reset, initialize } from "redux-form";

import { ContextMenuActionSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, CopyIcon, MoveToIcon } from "~/components/icon/icon";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveProjectDialog } from '~/store/projects/project-move-actions';
import { PROJECT_CREATE_FORM_NAME, openProjectCreateDialog } from '~/store/projects/project-create-actions';
import { openProjectUpdateDialog } from '~/store/projects/project-update-actions';

export const projectActionSet: ContextMenuActionSet = [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch(reset(PROJECT_CREATE_FORM_NAME));
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
