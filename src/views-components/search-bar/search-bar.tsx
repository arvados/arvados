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
    loadRecentQueries,
    openSearchView,
    closeSearchView,
    closeAdvanceView,
    navigateToItem,
    editSavedQuery,
    changeData,
    submitData, moveUp, moveDown, setAdvancedDataFromSearchValue, SEARCH_BAR_ADVANCED_FORM_NAME
} from '~/store/search-bar/search-bar-actions';
import { SearchBarView, SearchBarActionProps, SearchBarDataProps } from '~/views-components/search-bar/search-bar-view';
import { SearchBarAdvancedFormData } from '~/models/search-bar';
import { Vocabulary } from '~/models/vocabulary';

const mapStateToProps = ({ searchBar, form }: RootState): SearchBarDataProps => {
    return {
        searchValue: searchBar.searchValue,
        currentView: searchBar.currentView,
        isPopoverOpen: searchBar.open,
        searchResults: searchBar.searchResults,
        selectedItem: searchBar.selectedItem,
        savedQueries: searchBar.savedQueries,
        tags: form[SEARCH_BAR_ADVANCED_FORM_NAME],
        saveQuery: form[SEARCH_BAR_ADVANCED_FORM_NAME] &&
            form[SEARCH_BAR_ADVANCED_FORM_NAME].values &&
            form[SEARCH_BAR_ADVANCED_FORM_NAME].values!.saveQuery
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SearchBarActionProps => ({
    onSearch: (valueSearch: string) => dispatch<any>(searchData(valueSearch)),
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => dispatch<any>(changeData(event.target.value)),
    onSetView: (currentView: string) => dispatch(goToView(currentView)),
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => dispatch<any>(submitData(event)),
    closeView: () => dispatch<any>(closeSearchView()),
    closeAdvanceView: () => dispatch<any>(closeAdvanceView()),
    loadRecentQueries: () => dispatch<any>(loadRecentQueries()),
    deleteSavedQuery: (id: number) => dispatch<any>(deleteSavedQuery(id)),
    openSearchView: () => dispatch<any>(openSearchView()),
    navigateTo: (uuid: string) => dispatch<any>(navigateToItem(uuid)),
    editSavedQuery: (data: SearchBarAdvancedFormData) => dispatch<any>(editSavedQuery(data)),
    moveUp: () => dispatch<any>(moveUp()),
    moveDown: () => dispatch<any>(moveDown()),
    setAdvancedDataFromSearchValue: (search: string, vocabulary: Vocabulary) => dispatch<any>(setAdvancedDataFromSearchValue(search, vocabulary))
});

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(SearchBarView);
