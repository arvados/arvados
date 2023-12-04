// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { IconType } from 'components/icon/icon';
import { ResourcesState } from 'store/resources/resources';
import { FavoritesState } from 'store/favorites/favorites-reducer';
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';
import { AddFavoriteIcon, AdvancedIcon, DetailsIcon, FolderSharedIcon, Link, OpenIcon, PublicFavoriteIcon, RemoveFavoriteIcon, ShareIcon } from 'components/icon/icon';
import { checkFavorite } from 'store/favorites/favorites-reducer';
import { toggleFavorite } from 'store/favorites/favorites-actions';
import { favoritePanelActions } from 'store/favorite-panel/favorite-panel-action';
import { copyToClipboardAction, openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openWebDavS3InfoDialog } from 'store/collections/collection-info-actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
import { togglePublicFavorite } from "store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { PublicFavoritesState } from 'store/public-favorites/public-favorites-reducer';

export enum MultiSelectMenuActionNames {
    ADD_TO_FAVORITES = 'Add to Favorites',
    ADD_TO_TRASH = 'Add to Trash',
    ADD_TO_PUBLIC_FAVORITES = 'Add to public favorites',
    API_DETAILS = 'API Details',
    COPY_AND_RERUN_PROCESS = 'Copy and re-run process',
    COPY_TO_CLIPBOARD = 'Copy to clipboard',
    DELETE_WORKFLOW = 'Delete Worflow',
    EDIT_COLLECTION = 'Edit collection',
    EDIT_PROJECT = 'Edit project',
    FREEZE_PROJECT = 'Freeze Project',
    MAKE_A_COPY = 'Make a copy',
    MOVE_TO = 'Move to',
    NEW_PROJECT = 'New project',
    OPEN_IN_NEW_TAB = 'Open in new tab',
    OPEN_W_3RD_PARTY_CLIENT = 'Open with 3rd party client',
    REMOVE = 'Remove',
    RUN_WORKFLOW = 'Run Workflow',
    SHARE = 'Share',
    VIEW_DETAILS = 'View details',
};

export type MultiSelectMenuAction = {
    name: string;
    icon: IconType;
    hasAlts: boolean;
    altName?: string;
    altIcon?: IconType;
    isForMulti: boolean;
    useAlts?: (uuid: string, iconProps: {resources: ResourcesState, favorites: FavoritesState, publicFavorites: PublicFavoritesState}) => boolean;
    execute(dispatch: Dispatch, resources: ContextMenuResource[], state?: any): void;
    adminOnly?: boolean;
};

export type MultiSelectMenuActionSet = MultiSelectMenuAction[][];

const { ADD_TO_FAVORITES, ADD_TO_PUBLIC_FAVORITES, OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT, SHARE } = MultiSelectMenuActionNames;

const msToggleFavoriteAction: MultiSelectMenuAction = {
    name: ADD_TO_FAVORITES,
    icon: AddFavoriteIcon,
    hasAlts: true,
    altName: 'Remove from Favorites',
    altIcon: RemoveFavoriteIcon,
    isForMulti: false,
    useAlts: (uuid, iconProps) => {
        return checkFavorite(uuid, iconProps.favorites);
    },
    execute: (dispatch, resources) => {
        dispatch<any>(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
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

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

const msViewDetailsAction: MultiSelectMenuAction  = {
    name: VIEW_DETAILS,
    icon: DetailsIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch) => {
        dispatch<any>(toggleDetailsPanel());
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

const msOpenWith3rdPartyClientAction: MultiSelectMenuAction  = {
    name: OPEN_W_3RD_PARTY_CLIENT,
    icon: FolderSharedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

const msShareAction: MultiSelectMenuAction  = {
    name: SHARE,
    icon: ShareIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openSharingDialog(resources[0].uuid));
    },
};

const msTogglePublicFavoriteAction: MultiSelectMenuAction = {
    name: ADD_TO_PUBLIC_FAVORITES,
    icon: PublicFavoriteIcon,
    hasAlts: true,
    altName: 'Remove from public favorites',
    altIcon: PublicFavoriteIcon,
    isForMulti: false,
    useAlts: (uuid: string, iconProps) => {
        return iconProps.publicFavorites[uuid] === true
    },
    execute: (dispatch, resources) => {
        dispatch<any>(togglePublicFavorite(resources[0])).then(() => {
            dispatch(publicFavoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const msCommonActionSet = [
    msToggleFavoriteAction,
    msOpenInNewTabMenuAction,
    msCopyToClipboardMenuAction,
    msViewDetailsAction,
    msAdvancedAction,
    msOpenWith3rdPartyClientAction,
    msShareAction,
    msTogglePublicFavoriteAction
];
