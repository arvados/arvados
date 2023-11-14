// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DynamicContextMenuActionSet, DynamicContextMenuAction } from "views-components/context-menu/context-menu-action-set";
import { MoveToIcon, Link } from "components/icon/icon";
import { openMoveProjectDialog } from "store/projects/project-move-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "store/trash/trash-actions";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { ToggleFavoriteAction } from "views-components/context-menu/actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { AddFavoriteIcon, RemoveFavoriteIcon } from "components/icon/icon";
import { RestoreFromTrashIcon, TrashIcon } from "components/icon/icon";


export const msToggleFavoriteAction: DynamicContextMenuAction = {
    name: "ToggleFavoriteAction",
    defaultText: 'Add to Favorites',
    altText: 'Remove from Favorites',
    defaultIcon: AddFavoriteIcon,
    altIcon: RemoveFavoriteIcon,

    execute: (dispatch, resources) => {
        dispatch<any>(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const msCopyToClipboardMenuAction = {
    icon: Link,
    name: "Copy to clipboard",
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const msMoveToAction = {
    icon: MoveToIcon,
    name: "Move to",
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const msToggleTrashAction: DynamicContextMenuAction = {
    name: "ToggleTrashAction",
    defaultText: 'Add to Trash',
    altText: 'Restore from Trash',
    defaultIcon: TrashIcon,
    altIcon: RestoreFromTrashIcon,
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch<any>(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!, resources.length > 1));
        }
    },
};

export const msProjectActionSet: DynamicContextMenuAction[][] = [[msCopyToClipboardMenuAction, msMoveToAction, msToggleTrashAction, msToggleFavoriteAction]];
