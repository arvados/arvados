// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { goToView, searchBarActions } from '~/store/search-bar/search-bar-actions';
import { SearchBarView } from '~/views-components/search-bar/search-bar-view';
import { saveRecentQuery, loadRecentQueries } from '~/store/search-bar/search-bar-actions';

const mapStateToProps = ({ searchBar }: RootState) => {
    return {
        currentView: searchBar.currentView,
        open: searchBar.open
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onSetView: (currentView: string) => dispatch(goToView(currentView)),
    openView: () => dispatch<any>(searchBarActions.OPEN_SEARCH_VIEW()),
    closeView: () => dispatch<any>(searchBarActions.CLOSE_SEARCH_VIEW()),
    saveQuery: (query: string) => dispatch<any>(saveRecentQuery(query)),
    loadQueries: () => dispatch<any>(loadRecentQueries())
});

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(SearchBarView);