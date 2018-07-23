// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { projectActions } from "../../../store/project/project-action";
import { ShareIcon, NewProjectIcon } from "../../../components/icon/icon";
import { FavoriteActionText, FavoriteActionIcon } from "./favorite-action";
import { favoritesActions, toggleFavorite } from "../../../store/favorites/favorites-actions";

export const projectActionSet: ContextMenuActionSet = [[{
    icon: NewProjectIcon,
    name: "New project",
    execute: (dispatch, resource) => {
        dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
    }
}, {
    name: FavoriteActionText,
    icon: FavoriteActionIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(toggleFavorite(resource.uuid));
    }
}]];
