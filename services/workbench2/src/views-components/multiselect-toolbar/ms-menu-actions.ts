// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { IconType } from 'components/icon/icon';
import { ResourcesState } from 'store/resources/resources';
import { FavoritesState } from 'store/favorites/favorites-reducer';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { AddFavoriteIcon, AdvancedIcon, DetailsIcon, OpenIcon, PublicFavoriteIcon, RemoveFavoriteIcon } from 'components/icon/icon';
import { checkFavorite } from 'store/favorites/favorites-reducer';
import { toggleFavorite } from 'store/favorites/favorites-actions';
import { favoritePanelActions } from 'store/favorite-panel/favorite-panel-action';
import { openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { ToggleFavoriteAction } from 'views-components/context-menu/actions/favorite-action';
import { TogglePublicFavoriteAction } from 'views-components/context-menu/actions/public-favorite-action';

export type MultiSelectMenuAction = {
    name: string;
    icon: IconType;
    hasAlts: boolean;
    altName?: string;
    altIcon?: IconType;
    isForMulti: boolean;
    useAlts?: (uuid: string | null, iconProps: {resources: ResourcesState, favorites: FavoritesState, publicFavorites: PublicFavoritesState}) => boolean;
    execute(dispatch: Dispatch, resources: ContextMenuResource[], state?: any): void;
    adminOnly?: boolean;
};

export type MultiSelectMenuActionSet = MultiSelectMenuAction[][];

const { ADD_TO_FAVORITES, ADD_TO_PUBLIC_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS } = ContextMenuActionNames;

const msToggleFavoriteAction: any = {
    name: ADD_TO_FAVORITES,
    component: ToggleFavoriteAction,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

const msTogglePublicFavoriteAction: any = {
    name: ADD_TO_PUBLIC_FAVORITES,
    component: TogglePublicFavoriteAction,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(togglePublicFavorite(resources[0])).then(() => {
            dispatch(publicFavoritePanelActions.REQUEST_ITEMS());
        });
    },
};

const msOpenInNewTabMenuAction: MultiSelectMenuAction  = {
    name: OPEN_IN_NEW_TAB,
    icon: OpenIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openInNewTabAction(resources[0]));
    },
};

const msViewDetailsAction: MultiSelectMenuAction  = {
    name: VIEW_DETAILS,
    icon: DetailsIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openDetailsPanel(resources[0].uuid));
    },
};

const msAdvancedAction: MultiSelectMenuAction  = {
    name: API_DETAILS,
    icon: AdvancedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const msCommonActionSet = [
    msToggleFavoriteAction,
    msOpenInNewTabMenuAction,
    msViewDetailsAction,
    msAdvancedAction,
    msTogglePublicFavoriteAction
];
