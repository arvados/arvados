// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import {
    goToView,
    searchData,
    deleteSavedQuery,
    saveRecentQuery,
    loadRecentQueries,
    saveQuery,
    openSearchView,
    closeSearchView,
    navigateToItem,
    editSavedQuery,
    searchDataOnEnter,
    changeData,
    submitData
} from '~/store/search-bar/search-bar-actions';
import { SearchBarView, SearchBarActionProps, SearchBarDataProps } from '~/views-components/search-bar/search-bar-view';
import { SearchBarAdvanceFormData } from '~/models/search-bar';

const mapStateToProps = ({ searchBar, form }: RootState): SearchBarDataProps => {
    return {
        searchValue: searchBar.searchValue,
        currentView: searchBar.currentView,
        isPopoverOpen: searchBar.open,
        searchResults: searchBar.searchResults,
        savedQueries: searchBar.savedQueries,
        tags: form.searchBarAdvanceFormName
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SearchBarActionProps => ({
    onSearch: (valueSearch: string) => dispatch<any>(searchData(valueSearch)),
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => dispatch<any>(changeData(event.target.value)),
    onSetView: (currentView: string) => dispatch(goToView(currentView)),
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => dispatch<any>(submitData(event)),
    closeView: () => dispatch<any>(closeSearchView()),
    loadRecentQueries: () => dispatch<any>(loadRecentQueries()),
    saveQuery: (data: SearchBarAdvanceFormData) => dispatch<any>(saveQuery(data)),
    deleteSavedQuery: (id: number) => dispatch<any>(deleteSavedQuery(id)),
    openSearchView: () => dispatch<any>(openSearchView()),
    navigateTo: (uuid: string) => dispatch<any>(navigateToItem(uuid)),
    editSavedQuery: (data: SearchBarAdvanceFormData) => dispatch<any>(editSavedQuery(data))
});

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(SearchBarView);