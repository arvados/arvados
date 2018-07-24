// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { projectActions } from "../../../store/project/project-action";
import { NewProjectIcon } from "../../../components/icon/icon";
import { ToggleFavoriteAction } from "./favorite-action";
import { toggleFavorite } from "../../../store/favorites/favorites-actions";

export const projectActionSet: ContextMenuActionSet = [[{
    icon: NewProjectIcon,
    name: "New project",
    execute: (dispatch, resource) => {
        dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
    }
}, {
    component: ToggleFavoriteAction,
    execute: (dispatch, resource) => {
        dispatch<any>(toggleFavorite(resource));
    }
}]];
