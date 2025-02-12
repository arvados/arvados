// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, MoveToIcon, DetailsIcon, AdvancedIcon, OpenIcon, Link, FolderSharedIcon, CopyIcon } from "components/icon/icon";
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
import { openDetailsPanel } from "store/details-panel/details-panel-action";
import { copyToClipboardAction, copyStringToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { ToggleLockAction } from "../actions/lock-action";
import { freezeProject, unfreezeProject } from "store/projects/project-lock-actions";

export const toggleFavoriteAction = {
    component: ToggleFavoriteAction,
    name: ContextMenuActionNames.ADD_TO_FAVORITES,
    execute: (dispatch, resources) => {
        dispatch(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const openInNewTabMenuAction = {
    icon: OpenIcon,
    name: ContextMenuActionNames.OPEN_IN_NEW_TAB,
    execute: (dispatch, resources) => {
        dispatch(openInNewTabAction(resources[0]));
    },
};

export const copyToClipboardMenuAction = {
    icon: Link,
    name: ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const copyUuidAction = {
    icon: CopyIcon,
    name: ContextMenuActionNames.COPY_UUID,
    execute: (dispatch, resources) => {
        dispatch(copyStringToClipboardAction(resources[0].uuid));
    },
};

export const viewDetailsAction = {
    icon: DetailsIcon,
    name: ContextMenuActionNames.VIEW_DETAILS,
    execute: (dispatch, resources) => {
        dispatch(openDetailsPanel(resources[0].uuid));
    },
};

export const advancedAction = {
    icon: AdvancedIcon,
    name: ContextMenuActionNames.API_DETAILS,
    execute: (dispatch, resources) => {
        dispatch(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const openWith3rdPartyClientAction = {
    icon: FolderSharedIcon,
    name: ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
    execute: (dispatch, resources) => {
        dispatch(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

export const editProjectAction = {
    icon: RenameIcon,
    name: ContextMenuActionNames.EDIT_PROJECT,
    execute: (dispatch, resources) => {
        dispatch(openProjectUpdateDialog(resources[0]));
    },
};

export const shareAction = {
    icon: ShareIcon,
    name: ContextMenuActionNames.SHARE,
    execute: (dispatch, resources) => {
        dispatch(openSharingDialog(resources[0].uuid));
    },
};

export const moveToAction = {
    icon: MoveToIcon,
    name: ContextMenuActionNames.MOVE_TO,
    isForMulti: true,
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const toggleTrashAction = {
    component: ToggleTrashAction,
    name: ContextMenuActionNames.MOVE_TO_TRASH,
    isForMulti: true,
    execute: (dispatch, resources) => {
        dispatch(toggleProjectTrashed(resources[0].uuid, resources[0].ownerUuid, resources[0].isTrashed!!, resources.length > 1));
    },
};

export const freezeProjectAction = {
    component: ToggleLockAction,
    name: ContextMenuActionNames.FREEZE_PROJECT,
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
    name: ContextMenuActionNames.NEW_PROJECT,
    execute: (dispatch, resources): void => {
        dispatch(openProjectCreateDialog(resources[0].uuid));
    },
};

export const readOnlyProjectActionSet: ContextMenuActionSet = [
    [toggleFavoriteAction, openInNewTabMenuAction, copyToClipboardMenuAction, copyUuidAction, viewDetailsAction, advancedAction, openWith3rdPartyClientAction],
];

export const filterGroupActionSet: ContextMenuActionSet = [
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
        copyUuidAction,
    ],
];

export const frozenActionSet: ContextMenuActionSet = [
    [
        shareAction,
        toggleFavoriteAction,
        openInNewTabMenuAction,
        copyToClipboardMenuAction,
        viewDetailsAction,
        advancedAction,
        openWith3rdPartyClientAction,
        freezeProjectAction,
        copyUuidAction,
    ],
];

export const projectActionSet: ContextMenuActionSet = [
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
        copyUuidAction,
    ],
];

export const writeableProjectActionSet = [
    [
        toggleFavoriteAction,
        openInNewTabMenuAction,
        copyToClipboardMenuAction,
        viewDetailsAction,
        advancedAction,
        openWith3rdPartyClientAction,
        editProjectAction,
        moveToAction,
        toggleTrashAction,
        newProjectAction,
        copyUuidAction,
    ],
];

export const manageableProjectActionSet = [
    [
        viewDetailsAction,
        openInNewTabMenuAction,
        copyUuidAction,
        shareAction,
        freezeProjectAction,
        toggleFavoriteAction,
        copyToClipboardMenuAction,
        openWith3rdPartyClientAction,
        advancedAction,
    ],
];

export const frozenManageableProjectActionSet = [
    [
        viewDetailsAction,
        openInNewTabMenuAction,
        copyUuidAction,
        shareAction,
        toggleFavoriteAction,
        copyToClipboardMenuAction,
        openWith3rdPartyClientAction,
        advancedAction,
        freezeProjectAction,
    ],
];
