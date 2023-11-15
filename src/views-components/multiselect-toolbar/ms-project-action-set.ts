// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuAction, MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-action-set";
import { MoveToIcon, Link } from "components/icon/icon";
import { openMoveProjectDialog } from "store/projects/project-move-actions";
import { toggleProjectTrashed } from "store/trash/trash-actions";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { toggleFavorite } from "store/favorites/favorites-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { AddFavoriteIcon, RemoveFavoriteIcon } from "components/icon/icon";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";
import { getResource } from "store/resources/resources";
import { checkFavorite } from "store/favorites/favorites-reducer";

export const msToggleFavoriteAction = {
    name: MultiSelectMenuActionNames.TOGGLE_FAVORITE_ACTION,
    defaultText: 'Add to Favorites',
    altText: 'Remove from Favorites',
    icon: AddFavoriteIcon,
    altIcon: RemoveFavoriteIcon,
    isDefault: (uuid, resources, favorites)=>{
        return !checkFavorite(uuid, favorites);
    },
    execute: (dispatch, resources) => {
        dispatch(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const msCopyToClipboardMenuAction = {
    icon: Link,
    name: MultiSelectMenuActionNames.COPY_TO_CLIPBOARD,
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const msMoveToAction = {
    icon: MoveToIcon,
    name: MultiSelectMenuActionNames.MOVE_TO,
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const msToggleTrashAction = {
    name: MultiSelectMenuActionNames.TOGGLE_TRASH_ACTION,
    defaultText: 'Add to Trash',
    altText: 'Restore from Trash',
    icon: TrashIcon,
    altIcon: RestoreFromTrashIcon,
    isDefault: (uuid, resources, favorites = []) => {
        return uuid ? !(getResource(uuid)(resources) as any).isTrashed : true;
    },
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!, resources.length > 1));
        }
    },
};

export const msProjectActionSet: MultiSelectMenuAction[][] = [[msCopyToClipboardMenuAction, msMoveToAction, msToggleTrashAction, msToggleFavoriteAction]];
