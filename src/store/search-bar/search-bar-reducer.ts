// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { searchBarActions, SearchBarActions } from '~/store/search-bar/search-bar-actions';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { SearchBarAdvanceFormData } from '~/models/search-bar';

type SearchResult = GroupContentsResource;

interface SearchBar {
    currentView: string;
    open: boolean;
    searchResults: SearchResult[];
    searchValue: string;
    savedQueries: SearchBarAdvanceFormData[];
    selectedItem: string;
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
    savedQueries: [],
    selectedItem: ''
};

export const searchBarReducer = (state = initialState, action: SearchBarActions): SearchBar =>
    searchBarActions.match(action, {
        SET_CURRENT_VIEW: currentView => ({
            ...state,
            currentView,
            open: true
        }),
        OPEN_SEARCH_VIEW: () => ({ ...state, open: true }),
        CLOSE_SEARCH_VIEW: () => ({ ...state, open: false }),
        SET_SEARCH_RESULTS: searchResults => ({
            ...state,
            searchResults,
            selectedItem: searchResults.length > 0
                ? searchResults.findIndex(r => r.uuid === state.selectedItem) >= 0
                    ? state.selectedItem
                    : state.searchValue
                : state.searchValue
        }),
        SET_SEARCH_VALUE: searchValue => ({
            ...state,
            searchValue,
            selectedItem: state.searchValue === state.selectedItem
                ? searchValue
                : state.selectedItem
        }),
        SET_SAVED_QUERIES: savedQueries => ({ ...state, savedQueries }),
        UPDATE_SAVED_QUERY: searchQuery => ({ ...state, savedQueries: searchQuery }),
        SET_SELECTED_ITEM: item => ({ ...state, selectedItem: item }),
        MOVE_UP: () => {
            let selectedItem = state.selectedItem;
            if (state.currentView === SearchView.AUTOCOMPLETE) {
                const idx = state.searchResults.findIndex(r => r.uuid === selectedItem);
                if (idx > 0) {
                    selectedItem = state.searchResults[idx - 1].uuid;
                } else {
                    selectedItem = state.searchValue;
                }
            }
            return {
                ...state,
                selectedItem
            };
        },
        MOVE_DOWN: () => {
            let selectedItem = state.selectedItem;
            if (state.currentView === SearchView.AUTOCOMPLETE) {
                const idx = state.searchResults.findIndex(r => r.uuid === selectedItem);
                if (idx >= 0 && idx < state.searchResults.length - 1) {
                    selectedItem = state.searchResults[idx + 1].uuid;
                } else if (idx < 0 && state.searchResults.length > 0) {
                    selectedItem = state.searchResults[0].uuid;
                }
            }
            return {
                ...state,
                selectedItem
            };
        },
        default: () => state
    });
