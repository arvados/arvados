// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { TogglePublicFavoriteAction } from "views-components/context-menu/actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

import {
    shareAction,
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
    freezeProjectAction,
    editProjectAction,
    moveToAction,
    toggleTrashAction,
    newProjectAction,
    copyUuidAction,
} from "views-components/context-menu/action-sets/project-action-set";

export const togglePublicFavoriteAction = {
    component: TogglePublicFavoriteAction,
    name: ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
    execute: (dispatch, resources) => {
        dispatch(togglePublicFavorite(resources[0])).then(() => {
            dispatch(publicFavoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const projectAdminActionSet: ContextMenuActionSet = [
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
        togglePublicFavoriteAction,
        copyUuidAction,
    ],
];

export const filterGroupAdminActionSet: ContextMenuActionSet = [
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
        togglePublicFavoriteAction,
        copyUuidAction,
    ],
];

export const frozenAdminActionSet: ContextMenuActionSet = [
    [
        shareAction,
        togglePublicFavoriteAction,
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
