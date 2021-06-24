// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from "common/unionize";
import { GroupContentsResource, GroupContentsResourcePrefix } from 'services/groups-service/groups-service';
import { Dispatch } from 'redux';
import { change, initialize, untouch } from 'redux-form';
import { RootState } from 'store/store';
import { initUserProject, treePickerActions } from 'store/tree-picker/tree-picker-actions';
import { ServiceRepository } from 'services/services';
import { FilterBuilder } from "services/api/filter-builder";
import { ResourceKind, RESOURCE_UUID_REGEX, COLLECTION_PDH_REGEX } from 'models/resource';
import { SearchView } from 'store/search-bar/search-bar-reducer';
import { navigateTo, navigateToSearchResults } from 'store/navigation/navigation-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { PropertyValue, SearchBarAdvancedFormData } from 'models/search-bar';
import * as _ from "lodash";
import { getModifiedKeysValues } from "common/objects";
import { activateSearchBarProject } from "store/search-bar/search-bar-tree-actions";
import { Session } from "models/session";
import { searchResultsPanelActions } from "store/search-results-panel/search-results-panel-actions";
import { ListResults } from "services/common-service/common-service";
import * as parser from './search-query/arv-parser';
import { Keywords } from './search-query/arv-parser';
import { Vocabulary, getTagKeyLabel, getTagValueLabel } from "models/vocabulary";

export const searchBarActions = unionize({
    SET_CURRENT_VIEW: ofType<string>(),
    OPEN_SEARCH_VIEW: ofType<{}>(),
    CLOSE_SEARCH_VIEW: ofType<{}>(),
    SET_SEARCH_RESULTS: ofType<GroupContentsResource[]>(),
    SET_SEARCH_VALUE: ofType<string>(),
    SET_SAVED_QUERIES: ofType<SearchBarAdvancedFormData[]>(),
    SET_RECENT_QUERIES: ofType<string[]>(),
    UPDATE_SAVED_QUERY: ofType<SearchBarAdvancedFormData[]>(),
    SET_SELECTED_ITEM: ofType<string>(),
    MOVE_UP: ofType<{}>(),
    MOVE_DOWN: ofType<{}>(),
    SELECT_FIRST_ITEM: ofType<{}>()
});

export type SearchBarActions = UnionOf<typeof searchBarActions>;

export const SEARCH_BAR_ADVANCED_FORM_NAME = 'searchBarAdvancedFormName';

export const SEARCH_BAR_ADVANCED_FORM_PICKER_ID = 'searchBarAdvancedFormPickerId';

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const goToView = (currentView: string) => searchBarActions.SET_CURRENT_VIEW(currentView);

export const saveRecentQuery = (query: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) =>
        services.searchService.saveRecentQuery(query);


export const loadRecentQueries = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const recentQueries = services.searchService.getRecentQueries();
        dispatch(searchBarActions.SET_RECENT_QUERIES(recentQueries));
        return recentQueries;
    };

export const searchData = (searchValue: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const currentView = getState().searchBar.currentView;
        dispatch(searchResultsPanelActions.CLEAR());
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        if (searchValue.length > 0) {
            dispatch<any>(searchGroups(searchValue, 5));
            if (currentView === SearchView.BASIC) {
                dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
                dispatch(navigateToSearchResults(searchValue));
            }
        }
    };

export const searchAdvancedData = (data: SearchBarAdvancedFormData) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch<any>(saveQuery(data));
        const searchValue = getState().searchBar.searchValue;
        dispatch(searchResultsPanelActions.CLEAR());
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(navigateToSearchResults(searchValue));
    };

export const setSearchValueFromAdvancedData = (data: SearchBarAdvancedFormData, prevData?: SearchBarAdvancedFormData) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const searchValue = getState().searchBar.searchValue;
        const value = getQueryFromAdvancedData({
            ...data,
            searchValue
        }, prevData);
        dispatch(searchBarActions.SET_SEARCH_VALUE(value));
    };

export const setAdvancedDataFromSearchValue = (search: string, vocabulary: Vocabulary) =>
    async (dispatch: Dispatch) => {
        const data = getAdvancedDataFromQuery(search, vocabulary);
        dispatch<any>(initialize(SEARCH_BAR_ADVANCED_FORM_NAME, data));
        if (data.projectUuid) {
            await dispatch<any>(activateSearchBarProject(data.projectUuid));
            dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID, id: data.projectUuid }));
        }
    };

const saveQuery = (data: SearchBarAdvancedFormData) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const savedQueries = services.searchService.getSavedQueries();
        if (data.saveQuery && data.queryName) {
            const filteredQuery = savedQueries.find(query => query.queryName === data.queryName);
            data.searchValue = getState().searchBar.searchValue;
            if (filteredQuery) {
                services.searchService.editSavedQueries(data);
                dispatch(searchBarActions.UPDATE_SAVED_QUERY(savedQueries));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Query has been successfully updated', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            } else {
                services.searchService.saveQuery(data);
                dispatch(searchBarActions.SET_SAVED_QUERIES(savedQueries));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Query has been successfully saved', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
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

export const editSavedQuery = (data: SearchBarAdvancedFormData) =>
    (dispatch: Dispatch<any>) => {
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.ADVANCED));
        dispatch(searchBarActions.SET_SEARCH_VALUE(getQueryFromAdvancedData(data)));
        dispatch<any>(initialize(SEARCH_BAR_ADVANCED_FORM_NAME, data));
    };

export const openSearchView = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const savedSearchQueries = services.searchService.getSavedQueries();
        dispatch(searchBarActions.SET_SAVED_QUERIES(savedSearchQueries));
        dispatch(loadRecentQueries());
        dispatch(searchBarActions.OPEN_SEARCH_VIEW());
        dispatch(searchBarActions.SELECT_FIRST_ITEM());
    };

export const closeSearchView = () =>
    (dispatch: Dispatch<any>) => {
        dispatch(searchBarActions.SET_SELECTED_ITEM(''));
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
    };

export const closeAdvanceView = () =>
    (dispatch: Dispatch<any>) => {
        dispatch(searchBarActions.SET_SEARCH_VALUE(''));
        dispatch(treePickerActions.DEACTIVATE_TREE_PICKER_NODE({ pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID }));
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
    };

export const navigateToItem = (uuid: string) =>
    (dispatch: Dispatch<any>) => {
        dispatch(searchBarActions.SET_SELECTED_ITEM(''));
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(navigateTo(uuid));
    };

export const changeData = (searchValue: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        const currentView = getState().searchBar.currentView;
        const searchValuePresent = searchValue.length > 0;

        if (currentView === SearchView.ADVANCED) {
            dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.AUTOCOMPLETE));
        } else if (searchValuePresent) {
            dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.AUTOCOMPLETE));
            dispatch(searchBarActions.SET_SELECTED_ITEM(searchValue));
        } else {
            dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
            dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
            dispatch(searchBarActions.SELECT_FIRST_ITEM());
        }
    };

export const submitData = (event: React.FormEvent<HTMLFormElement>) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        event.preventDefault();
        const searchValue = getState().searchBar.searchValue;
        dispatch<any>(saveRecentQuery(searchValue));
        dispatch<any>(loadRecentQueries());
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        if (RESOURCE_UUID_REGEX.exec(searchValue) || COLLECTION_PDH_REGEX.exec(searchValue)) {
            dispatch<any>(navigateTo(searchValue));
        } else {
            dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
            dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
            dispatch(searchResultsPanelActions.CLEAR());
            dispatch(navigateToSearchResults(searchValue));
        }
    };


const searchGroups = (searchValue: string, limit: number) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentView = getState().searchBar.currentView;

        if (searchValue || currentView === SearchView.ADVANCED) {
            const { cluster: clusterId } = getAdvancedDataFromQuery(searchValue);
            const sessions = getSearchSessions(clusterId, getState().auth.sessions);
            const lists: ListResults<GroupContentsResource>[] = await Promise.all(sessions.map(session => {
                const filters = queryToFilters(searchValue, session.apiRevision);
                return services.groupsService.contents('', {
                    filters,
                    limit,
                    recursive: true
                }, session);
            }));

            const items = lists.reduce((items, list) => items.concat(list.items), [] as GroupContentsResource[]);
            dispatch(searchBarActions.SET_SEARCH_RESULTS(items));
        }
    };

const buildQueryFromKeyMap = (data: any, keyMap: string[][]) => {
    let value = data.searchValue;

    const addRem = (field: string, key: string) => {
        const v = data[key];
        // Remove previous search expression.
        if (data.hasOwnProperty(key)) {
            let pattern: string;
            if (v === false) {
                pattern = `${field.replace(':', '\\:\\s*')}\\s*`;
            } else if (key.startsWith('prop-')) {
                // On properties, only remove key:value duplicates, allowing
                // multiple properties with the same key.
                const oldValue = key.slice(5).split(':')[1];
                pattern = `${field.replace(':', '\\:\\s*')}\\:\\s*${oldValue}\\s*`;
            } else {
                pattern = `${field.replace(':', '\\:\\s*')}\\:\\s*[\\w|\\#|\\-|\\/]*\\s*`;
            }
            value = value.replace(new RegExp(pattern), '');
        }
        // Re-add it with the current search value.
        if (v) {
            const nv = v === true
                ? `${field}`
                : `${field}:${v}`;
            // Always append to the end to keep user-entered text at the start.
            value = value + ' ' + nv;
        }
    };
    keyMap.forEach(km => addRem(km[0], km[1]));
    return value;
};

export const getQueryFromAdvancedData = (data: SearchBarAdvancedFormData, prevData?: SearchBarAdvancedFormData) => {
    let value = '';

    const flatData = (data: SearchBarAdvancedFormData) => {
        const fo = {
            searchValue: data.searchValue,
            type: data.type,
            cluster: data.cluster,
            projectUuid: data.projectUuid,
            inTrash: data.inTrash,
            pastVersions: data.pastVersions,
            dateFrom: data.dateFrom,
            dateTo: data.dateTo,
        };
        (data.properties || []).forEach(p =>
            fo[`prop-"${p.keyID || p.key}":"${p.valueID || p.value}"`] = `"${p.valueID || p.value}"`
            );
        return fo;
    };

    const keyMap = [
        ['type', 'type'],
        ['cluster', 'cluster'],
        ['project', 'projectUuid'],
        [`is:${parser.States.TRASHED}`, 'inTrash'],
        [`is:${parser.States.PAST_VERSION}`, 'pastVersions'],
        ['from', 'dateFrom'],
        ['to', 'dateTo']
    ];
    _.union(data.properties, prevData ? prevData.properties : [])
        .forEach(p => keyMap.push(
            [`has:"${p.keyID || p.key}"`, `prop-"${p.keyID || p.key}":"${p.valueID || p.value}"`]
        ));

    const modified = getModifiedKeysValues(flatData(data), prevData ? flatData(prevData):{});
    value = buildQueryFromKeyMap(
        {searchValue: data.searchValue, ...modified} as SearchBarAdvancedFormData, keyMap);

    value = value.trim();
    return value;
};

export const getAdvancedDataFromQuery = (query: string, vocabulary?: Vocabulary): SearchBarAdvancedFormData => {
    const { tokens, searchString } = parser.parseSearchQuery(query);
    const getValue = parser.getValue(tokens);
    return {
        searchValue: searchString,
        type: getValue(Keywords.TYPE) as ResourceKind,
        cluster: getValue(Keywords.CLUSTER),
        projectUuid: getValue(Keywords.PROJECT),
        inTrash: parser.isTrashed(tokens),
        pastVersions: parser.isPastVersion(tokens),
        dateFrom: getValue(Keywords.FROM) || '',
        dateTo: getValue(Keywords.TO) || '',
        properties: vocabulary
            ? parser.getProperties(tokens).map(
                p => {
                    return {
                        keyID: p.key,
                        key: getTagKeyLabel(p.key, vocabulary),
                        valueID: p.value,
                        value: getTagValueLabel(p.key, p.value, vocabulary),
                    };
                })
            : parser.getProperties(tokens),
        saveQuery: false,
        queryName: ''
    };
};

export const getSearchSessions = (clusterId: string | undefined, sessions: Session[]): Session[] => {
    return sessions.filter(s => s.loggedIn && (!clusterId || s.clusterId === clusterId));
};

export const queryToFilters = (query: string, apiRevision: number) => {
    const data = getAdvancedDataFromQuery(query);
    const filter = new FilterBuilder();
    const resourceKind = data.type;

    if (data.searchValue) {
        filter.addFullTextSearch(data.searchValue);
    }

    if (data.projectUuid) {
        filter.addEqual('owner_uuid', data.projectUuid);
    }

    if (data.dateFrom) {
        filter.addGte('modified_at', buildDateFilter(data.dateFrom));
    }

    if (data.dateTo) {
        filter.addLte('modified_at', buildDateFilter(data.dateTo));
    }

    data.properties.forEach(p => {
        if (p.value) {
            if (apiRevision < 20200212) {
                filter
                    .addEqual(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.PROJECT)
                    .addEqual(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.COLLECTION)
                    .addEqual(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.PROCESS);
            } else {
                filter
                    .addContains(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.PROJECT)
                    .addContains(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.COLLECTION)
                    .addContains(`properties.${p.key}`, p.value, GroupContentsResourcePrefix.PROCESS);
            }
        }
        filter.addExists(p.key);
    });

    return filter
        .addIsA("uuid", buildUuidFilter(resourceKind))
        .getFilters();
};

const buildUuidFilter = (type?: ResourceKind): ResourceKind[] => {
    return type ? [type] : [ResourceKind.PROJECT, ResourceKind.COLLECTION, ResourceKind.PROCESS];
};

const buildDateFilter = (date?: string): string => {
    return date ? date : '';
};

export const initAdvancedFormProjectsTree = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(initUserProject(SEARCH_BAR_ADVANCED_FORM_PICKER_ID));
    };

export const changeAdvancedFormProperty = (propertyField: string, value: PropertyValue[] | string = '') =>
    (dispatch: Dispatch) => {
        dispatch(change(SEARCH_BAR_ADVANCED_FORM_NAME, propertyField, value));
    };

export const resetAdvancedFormProperty = (propertyField: string) =>
    (dispatch: Dispatch) => {
        dispatch(change(SEARCH_BAR_ADVANCED_FORM_NAME, propertyField, null));
        dispatch(untouch(SEARCH_BAR_ADVANCED_FORM_NAME, propertyField));
    };

export const moveUp = () =>
    (dispatch: Dispatch) => {
        dispatch(searchBarActions.MOVE_UP());
    };

export const moveDown = () =>
    (dispatch: Dispatch) => {
        dispatch(searchBarActions.MOVE_DOWN());
    };
