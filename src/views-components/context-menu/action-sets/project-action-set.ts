// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, MoveToIcon, DetailsIcon, AdvancedIcon, OpenIcon, Link, FolderSharedIcon } from 'components/icon/icon';
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveProjectDialog } from 'store/projects/project-move-actions';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';
import { openProjectUpdateDialog } from 'store/projects/project-update-actions';
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "store/trash/trash-actions";
import { ShareIcon } from 'components/icon/icon';
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { ToggleLockAction } from "../actions/lock-action";
import { freezeProject, unfreezeProject } from "store/projects/project-lock-actions";

export const toggleFavoriteAction = {
    component: ToggleFavoriteAction,
    name: 'ToggleFavoriteAction',
    execute: (dispatch, resource) => {
        dispatch(toggleFavorite(resource)).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    }
};

export const openInNewTabMenuAction = {
    icon: OpenIcon,
    name: "Open in new tab",
    execute: (dispatch, resource) => {
        dispatch(openInNewTabAction(resource));
    }
};

export const copyToClipboardMenuAction = {
    icon: Link,
    name: "Copy to clipboard",
    execute: (dispatch, resource) => {
        dispatch(copyToClipboardAction(resource));
    }
};

export const viewDetailsAction = {
    icon: DetailsIcon,
    name: "View details",
    execute: dispatch => {
        dispatch(toggleDetailsPanel());
    }
}

export const advancedAction = {
    icon: AdvancedIcon,
    name: "Advanced",
    execute: (dispatch, resource) => {
        dispatch(openAdvancedTabDialog(resource.uuid));
    }
}

export const openWith3rdPartyClientAction = {
    icon: FolderSharedIcon,
    name: "Open with 3rd party client",
    execute: (dispatch, resource) => {
        dispatch(openWebDavS3InfoDialog(resource.uuid));
    }
}

export const editProjectAction = {
    icon: RenameIcon,
    name: "Edit project",
    execute: (dispatch, resource) => {
        dispatch(openProjectUpdateDialog(resource));
    }
}

export const shareAction = {
    icon: ShareIcon,
    name: "Share",
    execute: (dispatch, { uuid }) => {
        dispatch(openSharingDialog(uuid));
    }
}

export const moveToAction = {
    icon: MoveToIcon,
    name: "Move to",
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource));
    }
}

export const toggleTrashAction = {
    component: ToggleTrashAction,
    name: 'ToggleTrashAction',
    execute: (dispatch, resource) => {
        dispatch(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!));
    }
}

export const freezeProjectAction = {
    component: ToggleLockAction,
    name: 'ToggleLockAction',
    execute: (dispatch, resource) => {
        if (resource.isFrozen) {
            dispatch(unfreezeProject(resource.uuid));
        } else {
            dispatch(freezeProject(resource.uuid));
        }

    }
}

export const newProjectAction: any = {
    icon: NewProjectIcon,
    name: "New project",
    execute: (dispatch, resource): void => {
        dispatch(openProjectCreateDialog(resource.uuid));
    }
}

export const readOnlyProjectActionSet: ContextMenuActionSet = [[
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
]];

export const filterGroupActionSet: ContextMenuActionSet = [[
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
    editProjectAction,
    shareAction,
    moveToAction,
    toggleTrashAction,
]];

export const frozenActionSet: ContextMenuActionSet = [[
    shareAction,
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
    freezeProjectAction
]];

export const projectActionSet: ContextMenuActionSet = [[
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
    editProjectAction,
    shareAction,
    moveToAction,
    toggleTrashAction,
    newProjectAction,
    freezeProjectAction,
]];
