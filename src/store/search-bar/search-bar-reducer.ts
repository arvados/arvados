// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { searchBarActions, SearchBarActions } from '~/store/search-bar/search-bar-actions';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { SearchBarAdvanceFormData } from '~/models/search-bar';

interface SearchBar {
    currentView: string;
    open: boolean;
    searchResults: GroupContentsResource[];
    searchValue: string;
    savedQueries: SearchBarAdvanceFormData[];
}

export enum SearchView {
    BASIC = 'basic',
    ADVANCED = 'advanced',
    AUTOCOMPLETE = 'autocomplete'
}

const initialState: SearchBar = {
    currentView: SearchView.BASIC,
    open: false,
    searchResults: [],
    searchValue: '',
    savedQueries: []
};

export const searchBarReducer = (state = initialState, action: SearchBarActions): SearchBar =>
    searchBarActions.match(action, {
        SET_CURRENT_VIEW: currentView => ({ ...state, currentView }),
        OPEN_SEARCH_VIEW: () => ({ ...state, open: true }),
        CLOSE_SEARCH_VIEW: () => ({ ...state, open: false }),
        SET_SEARCH_RESULTS: (searchResults) => ({ ...state, searchResults }),
        SET_SEARCH_VALUE: (searchValue) => ({ ...state, searchValue }),
        SET_SAVED_QUERIES: savedQueries => ({ ...state, savedQueries }),
        default: () => state
    });