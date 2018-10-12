// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import {
    goToView,
    searchData,
    searchBarActions,
    deleteSavedQuery,
    saveRecentQuery,
    loadRecentQueries,
    saveQuery,
    openSearchView
} from '~/store/search-bar/search-bar-actions';
import { SearchBarView } from '~/views-components/search-bar/search-bar-view';
import { SearchBarAdvanceFormData } from '~/models/search-bar';

const mapStateToProps = ({ searchBar }: RootState) => {
    return {
        searchValue: searchBar.searchValue,
        currentView: searchBar.currentView,
        isPopoverOpen: searchBar.open,
        searchResults: searchBar.searchResults,
        savedQueries: searchBar.savedQueries
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onSearch: (valueSearch: string) => dispatch<any>(searchData(valueSearch)),
    onSetView: (currentView: string) => dispatch(goToView(currentView)),
    closeView: () => dispatch<any>(searchBarActions.CLOSE_SEARCH_VIEW()),
    saveRecentQuery: (query: string) => dispatch<any>(saveRecentQuery(query)),
    loadRecentQueries: () => dispatch<any>(loadRecentQueries()),
    saveQuery: (data: SearchBarAdvanceFormData) => dispatch<any>(saveQuery(data)),
    deleteSavedQuery: (id: number) => dispatch<any>(deleteSavedQuery(id)),
    openSearchView: () => dispatch<any>(openSearchView())
});

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(SearchBarView);