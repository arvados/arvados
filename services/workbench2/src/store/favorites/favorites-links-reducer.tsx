// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from "models/link";

type FavoritesLinksState = LinkResource[];

const SET_FAVORITES_LINKS = 'SET_FAVORITES_LINKS';

export const favoritesLinksActions = {
    setFavoritesLinks: (links: LinkResource[]) => ({ type: SET_FAVORITES_LINKS, payload: links })
}

type FavoritesLinksAction = {
    type: string;
    payload: LinkResource[];
}

export const favoritesLinksReducer = (state: FavoritesLinksState = [], action: FavoritesLinksAction) => {
    switch (action.type) {
        case SET_FAVORITES_LINKS:
            return action.payload;
        default:
            return state;
    }
}
