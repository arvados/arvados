// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, MoveToIcon, DetailsIcon, AdvancedIcon, OpenIcon, Link, FolderSharedIcon } from "components/icon/icon";
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "store/favorites/favorites-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { openMoveProjectDialog } from "store/projects/project-move-actions";
import { openProjectCreateDialog } from "store/projects/project-create-actions";
import { openProjectUpdateDialog } from "store/projects/project-update-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "store/trash/trash-actions";
import { ShareIcon } from "components/icon/icon";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from "store/details-panel/details-panel-action";
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { ToggleLockAction } from "../actions/lock-action";
import { freezeProject, unfreezeProject } from "store/projects/project-lock-actions";

export const toggleFavoriteAction = {
    component: ToggleFavoriteAction,
    name: "ToggleFavoriteAction",
    execute: (dispatch, resources) => {
        dispatch(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const openInNewTabMenuAction = {
    icon: OpenIcon,
    name: "Open in new tab",
    execute: (dispatch, resources) => {
        dispatch(openInNewTabAction(resources[0]));
    },
};

export const copyToClipboardMenuAction = {
    icon: Link,
    name: "Copy to clipboard",
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const viewDetailsAction = {
    icon: DetailsIcon,
    name: "View details",
    execute: dispatch => {
        dispatch(toggleDetailsPanel());
    },
};

export const advancedAction = {
    icon: AdvancedIcon,
    name: "API Details",
    execute: (dispatch, resources) => {
        dispatch(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const openWith3rdPartyClientAction = {
    icon: FolderSharedIcon,
    name: "Open with 3rd party client",
    execute: (dispatch, resources) => {
        dispatch(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

export const editProjectAction = {
    icon: RenameIcon,
    name: "Edit project",
    execute: (dispatch, resources) => {
        dispatch(openProjectUpdateDialog(resources[0]));
    },
};

export const shareAction = {
    icon: ShareIcon,
    name: "Share",
    execute: (dispatch, resources) => {
        dispatch(openSharingDialog(resources[0].uuid));
    },
};

export const moveToAction = {
    icon: MoveToIcon,
    name: "Move to",
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const toggleTrashAction = {
    component: ToggleTrashAction,
    name: "ToggleTrashAction",
    execute: (dispatch, resources) => {
        dispatch(toggleProjectTrashed(resources[0].uuid, resources[0].ownerUuid, resources[0].isTrashed!!, resources.length > 1));
    },
};

export const freezeProjectAction = {
    component: ToggleLockAction,
    name: "ToggleLockAction",
    execute: (dispatch, resources) => {
        if (resources[0].isFrozen) {
            dispatch(unfreezeProject(resources[0].uuid));
        } else {
            dispatch(freezeProject(resources[0].uuid));
        }
    },
};

export const newProjectAction: any = {
    icon: NewProjectIcon,
    name: "New project",
    execute: (dispatch, resource): void => {
        dispatch(openProjectCreateDialog(resource.uuid));
    },
};

export const readOnlyProjectActionSet: ContextMenuActionItemSet = [
    [toggleFavoriteAction, openInNewTabMenuAction, copyToClipboardMenuAction, viewDetailsAction, advancedAction, openWith3rdPartyClientAction],
];

export const filterGroupActionSet: ContextMenuActionItemSet = [
    [
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
    ],
];

export const frozenActionSet: ContextMenuActionItemSet = [
    [
        shareAction,
        toggleFavoriteAction,
        openInNewTabMenuAction,
        copyToClipboardMenuAction,
        viewDetailsAction,
        advancedAction,
        openWith3rdPartyClientAction,
        freezeProjectAction,
    ],
];

export const projectActionSet: ContextMenuActionItemSet = [
    [
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
    ],
];
