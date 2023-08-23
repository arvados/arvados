// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    getQueryFromAdvancedData,
    searchBarActions,
    SearchBarActions
} from 'store/search-bar/search-bar-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { SearchBarAdvancedFormData } from 'models/search-bar';

type SearchResult = GroupContentsResource;
export type SearchBarSelectedItem = {
    id: string,
    query: string
};

interface SearchBar {
    currentView: string;
    open: boolean;
    searchResults: SearchResult[];
    searchValue: string;
    savedQueries: SearchBarAdvancedFormData[];
    recentQueries: string[];
    selectedItem: SearchBarSelectedItem;
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
    recentQueries: [],
    selectedItem: {
        id: '',
        query: ''
    }
};

const makeSelectedItem = (id: string, query?: string): SearchBarSelectedItem => ({ id, query: query ? query : id });

const makeQueryList = (recentQueries: string[], savedQueries: SearchBarAdvancedFormData[]) => {
    const recentIds = recentQueries.map((q, idx) => makeSelectedItem(`RQ-${idx}-${q}`, q));
    const savedIds = savedQueries.map((q, idx) => makeSelectedItem(`SQ-${idx}-${q.queryName}`, getQueryFromAdvancedData(q)));
    return recentIds.concat(savedIds);
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
            selectedItem: makeSelectedItem(searchResults.length > 0
                ? searchResults.findIndex(r => r.uuid === state.selectedItem.id) >= 0
                    ? state.selectedItem.id
                    : state.searchValue
                : state.searchValue
            )
        }),
        SET_SEARCH_VALUE: searchValue => ({
            ...state,
            searchValue
        }),
        SET_SAVED_QUERIES: savedQueries => ({ ...state, savedQueries }),
        SET_RECENT_QUERIES: recentQueries => ({ ...state, recentQueries }),
        UPDATE_SAVED_QUERY: searchQuery => ({ ...state, savedQueries: searchQuery }),
        SET_SELECTED_ITEM: item => ({ ...state, selectedItem: makeSelectedItem(item) }),
        MOVE_UP: () => {
            let selectedItem = state.selectedItem;
            if (state.currentView === SearchView.AUTOCOMPLETE) {
                const idx = state.searchResults.findIndex(r => r.uuid === selectedItem.id);
                if (idx > 0) {
                    selectedItem = makeSelectedItem(state.searchResults[idx - 1].uuid);
                } else {
                    selectedItem = makeSelectedItem(state.searchValue);
                }
            } else if (state.currentView === SearchView.BASIC) {
                const items = makeQueryList(state.recentQueries, state.savedQueries);

                const idx = items.findIndex(i => i.id === selectedItem.id);
                if (idx > 0) {
                    selectedItem = items[idx - 1];
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
                const idx = state.searchResults.findIndex(r => r.uuid === selectedItem.id);
                if (idx >= 0 && idx < state.searchResults.length - 1) {
                    selectedItem = makeSelectedItem(state.searchResults[idx + 1].uuid);
                } else if (idx < 0 && state.searchResults.length > 0) {
                    selectedItem = makeSelectedItem(state.searchResults[0].uuid);
                }
            } else if (state.currentView === SearchView.BASIC) {
                const items = makeQueryList(state.recentQueries, state.savedQueries);

                const idx = items.findIndex(i => i.id === selectedItem.id);
                if (idx >= 0 && idx < items.length - 1) {
                    selectedItem = items[idx + 1];
                }

                if (idx < 0 && items.length > 0) {
                    selectedItem = items[0];
                }
            }
            return {
                ...state,
                selectedItem
            };
        },
        SELECT_FIRST_ITEM: () => {
            let selectedItem = state.selectedItem;
            if (state.currentView === SearchView.AUTOCOMPLETE) {
                selectedItem = makeSelectedItem(state.searchValue);
            } else if (state.currentView === SearchView.BASIC) {
                const items = makeQueryList(state.recentQueries, state.savedQueries);
                if (items.length > 0) {
                    selectedItem = items[0];
                }
            }
            return {
                ...state,
                selectedItem
            };
        },
        default: () => state
    });
