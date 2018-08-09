// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reset } from "redux-form";

import { ContextMenuActionSet } from "../context-menu-action-set";
import { projectActions } from "../../../store/project/project-action";
import { NewProjectIcon, MoveToIcon } from "../../../components/icon/icon";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "../../../store/favorites/favorites-actions";
import { favoritePanelActions } from "../../../store/favorite-panel/favorite-panel-action";
import { openMoveToDialog } from "../../move-to-dialog/move-to-dialog";
import { PROJECT_CREATE_DIALOG } from "../../dialog-create/dialog-project-create";

export const projectActionSet: ContextMenuActionSet = [[{
    icon: NewProjectIcon,
    name: "New project",
    execute: (dispatch, resource) => {
        dispatch(reset(PROJECT_CREATE_DIALOG));
        dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
    }
}, {
    component: ToggleFavoriteAction,
    execute: (dispatch, resource) => {
        dispatch<any>(toggleFavorite(resource)).then(() => {
            dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
        });
    }
}, {
    icon: MoveToIcon,
    name: "Move to",
    execute: (dispatch) => {
        dispatch<any>(openMoveToDialog());
    }
},]];
