// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { searchBarActions, SearchBarActions } from '~/store/search-bar/search-bar-actions';

interface SearchBar {
    currentView: string;
    open: boolean;
}

export enum SearchView {
    BASIC = 'basic',
    ADVANCED = 'advanced',
    AUTOCOMPLETE = 'autocomplete'
}

const initialState: SearchBar = {
    currentView: SearchView.BASIC,
    open: false
};

export const searchBarReducer = (state = initialState, action: SearchBarActions): SearchBar =>
    searchBarActions.match(action, {
        SET_CURRENT_VIEW: currentView => ({ ...state, currentView }),
        OPEN_SEARCH_VIEW: () => ({ ...state, open: true }),
        CLOSE_SEARCH_VIEW: () => ({ ...state, open: false }),
        default: () => state
    });