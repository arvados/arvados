// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { TogglePublicFavoriteAction } from "views-components/context-menu/actions/public-favorite-action";
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";

import { shareAction, toggleFavoriteAction, openInNewTabMenuAction, copyToClipboardMenuAction, viewDetailsAction, advancedAction, openWith3rdPartyClientAction, freezeProjectAction, editProjectAction, moveToAction, toggleTrashAction, newProjectAction } from "views-components/context-menu/action-sets/project-action-set";

export const togglePublicFavoriteAction = {
    component: TogglePublicFavoriteAction,
    name: 'TogglePublicFavoriteAction',
    execute: (dispatch, resource) => {
        dispatch(togglePublicFavorite(resource)).then(() => {
            dispatch(publicFavoritePanelActions.REQUEST_ITEMS());
        });
}}

export const projectAdminActionSet: ContextMenuActionSet = [[
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
    togglePublicFavoriteAction
]];

export const filterGroupAdminActionSet: ContextMenuActionSet = [[
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
    togglePublicFavoriteAction
]];


export const frozenAdminActionSet: ContextMenuActionSet = [[
    shareAction,
    togglePublicFavoriteAction,
    toggleFavoriteAction,
    openInNewTabMenuAction,
    copyToClipboardMenuAction,
    viewDetailsAction,
    advancedAction,
    openWith3rdPartyClientAction,
    freezeProjectAction
]];
