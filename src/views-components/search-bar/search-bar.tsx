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
    saveQuery
} from '~/store/search-bar/search-bar-actions';
import { SearchBarView } from '~/views-components/search-bar/search-bar-view';

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
    openView: () => dispatch<any>(searchBarActions.OPEN_SEARCH_VIEW()),
    closeView: () => dispatch<any>(searchBarActions.CLOSE_SEARCH_VIEW()),
    saveRecentQuery: (query: string) => dispatch<any>(saveRecentQuery(query)),
    loadRecentQueries: () => dispatch<any>(loadRecentQueries()),
    saveQuery: (query: string) => dispatch<any>(saveQuery(query)),
    deleteSavedQuery: (id: number) => dispatch<any>(deleteSavedQuery(id))
});

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(SearchBarView);