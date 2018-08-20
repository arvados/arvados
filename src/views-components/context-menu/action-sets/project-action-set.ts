// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reset, initialize } from "redux-form";

import { ContextMenuActionSet } from "../context-menu-action-set";
import { projectActions, PROJECT_FORM_NAME } from "~/store/project/project-action";
import { NewProjectIcon, RenameIcon, CopyIcon } from "~/components/icon/icon";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { PROJECT_CREATE_DIALOG } from "../../dialog-create/dialog-project-create";
import { openMakeACopyDialog } from "~/views-components/make-a-copy-dialog/make-a-copy-dialog";

export const projectActionSet: ContextMenuActionSet = [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch(reset(PROJECT_CREATE_DIALOG));
            dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
        }
    },
    {
        icon: RenameIcon,
        name: "Edit project",
        execute: (dispatch, resource) => {
            dispatch(projectActions.OPEN_PROJECT_UPDATER({ uuid: resource.uuid }));
            dispatch(initialize(PROJECT_FORM_NAME, { name: resource.name, description: resource.description }));
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
        execute: dispatch => dispatch<any>(openMakeACopyDialog())
    },
]];
