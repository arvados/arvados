// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { GroupContentsResource, GroupContentsResourcePrefix } from '~/services/groups-service/groups-service';
import { Dispatch } from 'redux';
import { change, arrayPush } from 'redux-form';
import { RootState } from '~/store/store';
import { initUserProject } from '~/store/tree-picker/tree-picker-actions';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from "~/services/api/filter-builder";
import { ResourceKind } from '~/models/resource';
import { GroupClass } from '~/models/group';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { navigateToSearchResults, navigateTo } from '~/store/navigation/navigation-action';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { initialize } from 'redux-form';
import { SearchBarAdvanceFormData, PropertyValues } from '~/models/search-bar';
import { debounce } from 'debounce';

export const searchBarActions = unionize({
    SET_CURRENT_VIEW: ofType<string>(),
    OPEN_SEARCH_VIEW: ofType<{}>(),
    CLOSE_SEARCH_VIEW: ofType<{}>(),
    SET_SEARCH_RESULTS: ofType<GroupContentsResource[]>(),
    SET_SEARCH_VALUE: ofType<string>(),
    SET_SAVED_QUERIES: ofType<SearchBarAdvanceFormData[]>(),
    UPDATE_SAVED_QUERY: ofType<SearchBarAdvanceFormData[]>()
});

export type SearchBarActions = UnionOf<typeof searchBarActions>;

export const SEARCH_BAR_ADVANCE_FORM_NAME = 'searchBarAdvanceFormName';

export const SEARCH_BAR_ADVANCE_FORM_PICKER_ID = 'searchBarAdvanceFormPickerId';

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const goToView = (currentView: string) => searchBarActions.SET_CURRENT_VIEW(currentView);

export const saveRecentQuery = (query: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) =>
        services.searchService.saveRecentQuery(query);


export const loadRecentQueries = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const recentSearchQueries = services.searchService.getRecentQueries();
        return recentSearchQueries || [];
    };

export const searchData = (searchValue: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentView = getState().searchBar.currentView;
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
        dispatch<any>(searchGroups(searchValue, 5, {}));
        if (currentView === SearchView.BASIC) {
            dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
            dispatch(navigateToSearchResults);
        }

    };

export const searchAdvanceData = (data: SearchBarAdvanceFormData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const searchValue = getState().searchBar.searchValue;
        dispatch<any>(saveQuery(data));
        dispatch<any>(searchGroups(searchValue, 100, data));
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(navigateToSearchResults);
    };

// Todo: create ids for particular searchQuery
const saveQuery = (data: SearchBarAdvanceFormData) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const savedSearchQueries = services.searchService.getSavedQueries();
        const filteredQuery = savedSearchQueries.find(query => query.searchQuery === data.searchQuery);
        if (data.saveQuery && data.searchQuery) {
            if (filteredQuery) {
                services.searchService.editSavedQueries(data);
                dispatch(searchBarActions.UPDATE_SAVED_QUERY(savedSearchQueries));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Query has been sucessfully updated', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            } else {
                services.searchService.saveQuery(data);
                dispatch(searchBarActions.SET_SAVED_QUERIES(savedSearchQueries));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Query has been sucessfully saved', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            }
        }
    };

export const deleteSavedQuery = (id: number) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        services.searchService.deleteSavedQuery(id);
        const savedSearchQueries = services.searchService.getSavedQueries();
        dispatch(searchBarActions.SET_SAVED_QUERIES(savedSearchQueries));
        return savedSearchQueries || [];
    };

export const editSavedQuery = (data: SearchBarAdvanceFormData) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.ADVANCED));
        dispatch(searchBarActions.SET_SEARCH_VALUE(data.searchQuery));
        dispatch<any>(initialize(SEARCH_BAR_ADVANCE_FORM_NAME, data));
    };

export const openSearchView = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.OPEN_SEARCH_VIEW());
        const savedSearchQueries = services.searchService.getSavedQueries();
        dispatch(searchBarActions.SET_SAVED_QUERIES(savedSearchQueries));
    };

export const closeSearchView = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const isOpen = getState().searchBar.open;
        if (isOpen) {
            dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
            dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
        }
    };

export const closeAdvanceView = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.SET_SEARCH_VALUE(''));
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
    };

export const navigateToItem = (uuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(navigateTo(uuid));
    };

export const changeData = (searchValue: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        const currentView = getState().searchBar.currentView;
        const searchValuePresent = searchValue.length > 0;

        if (currentView === SearchView.ADVANCED) {

        } else if (searchValuePresent) {
            dispatch<any>(goToView(SearchView.AUTOCOMPLETE));
            debounceStartSearch(dispatch);
        } else {
            dispatch<any>(goToView(SearchView.BASIC));
            dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
            dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
        }
    };

export const submitData = (event: React.FormEvent<HTMLFormElement>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        event.preventDefault();
        const searchValue = getState().searchBar.searchValue;
        dispatch<any>(saveRecentQuery(searchValue));
        dispatch<any>(searchDataOnEnter(searchValue));
        dispatch<any>(loadRecentQueries());
    };

const searchDataOnEnter = (searchValue: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
        dispatch<any>(searchGroups(searchValue, 100, {}));
        dispatch(navigateToSearchResults);
    };

const debounceStartSearch = debounce((dispatch: Dispatch) => dispatch<any>(startSearch()), DEFAULT_SEARCH_DEBOUNCE);

const startSearch = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const searchValue = getState().searchBar.searchValue;
        dispatch<any>(searchData(searchValue));
    };

const searchGroups = (searchValue: string, limit: number, {...props}) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentView = getState().searchBar.currentView;

        if (searchValue || currentView === SearchView.ADVANCED) {
            const filters = getFilters('name', searchValue, props);
            const { items } = await services.groupsService.contents('', {
                filters,
                limit,
                recursive: true
            });
            dispatch(searchBarActions.SET_SEARCH_RESULTS(items));
        }
    };

export const getFilters = (filterName: string, searchValue: string, {...props}): string => {
    const { resourceKind, dateTo, dateFrom } = props;
    return new FilterBuilder()
        .addIsA("uuid", buildUuidFilter(resourceKind))
        .addILike(filterName, searchValue, GroupContentsResourcePrefix.COLLECTION)
        .addILike(filterName, searchValue, GroupContentsResourcePrefix.PROCESS)
        .addILike(filterName, searchValue, GroupContentsResourcePrefix.PROJECT)
        .addLte('created_at', buildDateFilter(dateTo))
        .addGte('created_at', buildDateFilter(dateFrom))
        .addEqual('groupClass', GroupClass.PROJECT, GroupContentsResourcePrefix.PROJECT)
        .getFilters();
};

const buildUuidFilter = (type?: ResourceKind): ResourceKind[] => {
    return type ? [type] : [ResourceKind.PROJECT, ResourceKind.COLLECTION, ResourceKind.PROCESS];
};

const buildDateFilter = (date?: string): string => {
    return date ? date : '';
};

export const initAdvanceFormProjectsTree = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(initUserProject(SEARCH_BAR_ADVANCE_FORM_PICKER_ID));
    };

export const changeAdvanceFormProperty = (property: string, value: PropertyValues[] | string = '') =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(change(SEARCH_BAR_ADVANCE_FORM_NAME, property, value));
    };

export const updateAdvanceFormProperties = (propertyValues: PropertyValues) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(arrayPush(SEARCH_BAR_ADVANCE_FORM_NAME, 'properties', propertyValues));
    };