// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize, UnionOf } from "~/common/unionize";
import { GroupContentsResource, GroupContentsResourcePrefix } from '~/services/groups-service/groups-service';
import { Dispatch } from 'redux';
import { arrayPush, change, initialize } from 'redux-form';
import { RootState } from '~/store/store';
import { initUserProject, treePickerActions } from '~/store/tree-picker/tree-picker-actions';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from "~/services/api/filter-builder";
import { ResourceKind } from '~/models/resource';
import { GroupClass } from '~/models/group';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { navigateTo, navigateToSearchResults } from '~/store/navigation/navigation-action';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { ClusterObjectType, PropertyValue, SearchBarAdvanceFormData } from '~/models/search-bar';
import { debounce } from 'debounce';
import * as _ from "lodash";
import { getModifiedKeysValues } from "~/common/objects";
import { activateSearchBarProject } from "~/store/search-bar/search-bar-tree-actions";

export const searchBarActions = unionize({
    SET_CURRENT_VIEW: ofType<string>(),
    OPEN_SEARCH_VIEW: ofType<{}>(),
    CLOSE_SEARCH_VIEW: ofType<{}>(),
    SET_SEARCH_RESULTS: ofType<GroupContentsResource[]>(),
    SET_SEARCH_VALUE: ofType<string>(),
    SET_SAVED_QUERIES: ofType<SearchBarAdvanceFormData[]>(),
    SET_RECENT_QUERIES: ofType<string[]>(),
    UPDATE_SAVED_QUERY: ofType<SearchBarAdvanceFormData[]>(),
    SET_SELECTED_ITEM: ofType<string>(),
    MOVE_UP: ofType<{}>(),
    MOVE_DOWN: ofType<{}>(),
    SELECT_FIRST_ITEM: ofType<{}>()
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
        const recentQueries = services.searchService.getRecentQueries();
        dispatch(searchBarActions.SET_RECENT_QUERIES(recentQueries));
        return recentQueries;
    };

export const searchData = (searchValue: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const currentView = getState().searchBar.currentView;
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        if (searchValue.length > 0) {
            dispatch<any>(searchGroups(searchValue, 5));
            if (currentView === SearchView.BASIC) {
                dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
                dispatch(navigateToSearchResults);
            }
        }
    };

export const searchAdvanceData = (data: SearchBarAdvanceFormData) =>
    async (dispatch: Dispatch) => {
        dispatch<any>(saveQuery(data));
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.BASIC));
        dispatch(searchBarActions.CLOSE_SEARCH_VIEW());
        dispatch(navigateToSearchResults);
    };

export const setSearchValueFromAdvancedData = (data: SearchBarAdvanceFormData, prevData?: SearchBarAdvanceFormData) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const searchValue = getState().searchBar.searchValue;
        const value = getQueryFromAdvancedData({
            ...data,
            searchValue
        }, prevData);
        dispatch(searchBarActions.SET_SEARCH_VALUE(value));
    };

export const setAdvancedDataFromSearchValue = (search: string) =>
    async (dispatch: Dispatch) => {
        const data = getAdvancedDataFromQuery(search);
        dispatch<any>(initialize(SEARCH_BAR_ADVANCE_FORM_NAME, data));
        if (data.projectUuid) {
            await dispatch<any>(activateSearchBarProject(data.projectUuid));
            dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ pickerId: SEARCH_BAR_ADVANCE_FORM_PICKER_ID, id: data.projectUuid }));
        }
    };

const saveQuery = (data: SearchBarAdvanceFormData) =>
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

export const editSavedQuery = (data: SearchBarAdvanceFormData) =>
    (dispatch: Dispatch<any>) => {
        dispatch(searchBarActions.SET_CURRENT_VIEW(SearchView.ADVANCED));
        dispatch(searchBarActions.SET_SEARCH_VALUE(getQueryFromAdvancedData(data)));
        dispatch<any>(initialize(SEARCH_BAR_ADVANCE_FORM_NAME, data));
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
        dispatch(treePickerActions.DEACTIVATE_TREE_PICKER_NODE({ pickerId: SEARCH_BAR_ADVANCE_FORM_PICKER_ID }));
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
            debounceStartSearch(dispatch);
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
        dispatch(searchBarActions.SET_SEARCH_VALUE(searchValue));
        dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
        dispatch(navigateToSearchResults);
    };

const debounceStartSearch = debounce((dispatch: Dispatch) => dispatch<any>(startSearch()), DEFAULT_SEARCH_DEBOUNCE);

const startSearch = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const searchValue = getState().searchBar.searchValue;
        dispatch<any>(searchData(searchValue));
    };

const searchGroups = (searchValue: string, limit: number) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentView = getState().searchBar.currentView;

        if (searchValue || currentView === SearchView.ADVANCED) {
            const filters = getFilters('name', searchValue);
            const { items } = await services.groupsService.contents('', {
                filters,
                limit,
                recursive: true
            });
            dispatch(searchBarActions.SET_SEARCH_RESULTS(items));
        }
    };

const buildQueryFromKeyMap = (data: any, keyMap: string[][], mode: 'rebuild' | 'reuse') => {
    let value = data.searchValue;

    const addRem = (field: string, key: string) => {
        const v = data[key];

        if (data.hasOwnProperty(key)) {
            const pattern = v === false
                ? `${field.replace(':', '\\:\\s*')}\\s*`
                : `${field.replace(':', '\\:\\s*')}\\:\\s*[\\w|\\#|\\-|\\/]*\\s*`;
            value = value.replace(new RegExp(pattern), '');
        }

        if (v) {
            const nv = v === true
                ? `${field}`
                : `${field}:${v}`;

            if (mode === 'rebuild') {
                value = value + ' ' + nv;
            } else {
                value = nv + ' ' + value;
            }
        }
    };

    keyMap.forEach(km => addRem(km[0], km[1]));

    return value;
};

export const getQueryFromAdvancedData = (data: SearchBarAdvanceFormData, prevData?: SearchBarAdvanceFormData) => {
    let value = '';

    const flatData = (data: SearchBarAdvanceFormData) => {
        const fo = {
            searchValue: data.searchValue,
            type: data.type,
            cluster: data.cluster,
            projectUuid: data.projectUuid,
            inTrash: data.inTrash,
            dateFrom: data.dateFrom,
            dateTo: data.dateTo,
        };
        (data.properties || []).forEach(p => fo[`prop-${p.key}`] = p.value);
        return fo;
    };

    const keyMap = [
        ['type', 'type'],
        ['cluster', 'cluster'],
        ['project', 'projectUuid'],
        ['is:trashed', 'inTrash'],
        ['from', 'dateFrom'],
        ['to', 'dateTo']
    ];
    _.union(data.properties, prevData ? prevData.properties : [])
        .forEach(p => keyMap.push([`has:${p.key}`, `prop-${p.key}`]));

    if (prevData) {
        const obj = getModifiedKeysValues(flatData(data), flatData(prevData));
        value = buildQueryFromKeyMap({
            searchValue: data.searchValue,
            ...obj
        } as SearchBarAdvanceFormData, keyMap, "reuse");
    } else {
        value = buildQueryFromKeyMap(flatData(data), keyMap, "rebuild");
    }

    value = value.trim();
    return value;
};

export interface ParseSearchQuery {
    hasKeywords: boolean;
    values: string[];
    properties: {
        [key: string]: string[]
    };
}

export const parseSearchQuery: (query: string) => ParseSearchQuery = (searchValue: string) => {
    const keywords = [
        'type:',
        'cluster:',
        'project:',
        'is:',
        'from:',
        'to:',
        'has:'
    ];

    const hasKeywords = (search: string) => keywords.reduce((acc, keyword) => acc + (search.includes(keyword) ? 1 : 0), 0);
    let keywordsCnt = 0;

    const properties = {};

    keywords.forEach(k => {
        let p = searchValue.indexOf(k);
        const key = k.substr(0, k.length - 1);

        while (p >= 0) {
            const l = searchValue.length;
            keywordsCnt += 1;

            let v = '';
            let i = p + k.length;
            while (i < l && searchValue[i] === ' ') {
                ++i;
            }
            const vp = i;
            while (i < l && searchValue[i] !== ' ') {
                v += searchValue[i];
                ++i;
            }

            if (hasKeywords(v)) {
                searchValue = searchValue.substr(0, p) + searchValue.substr(vp);
            } else {
                if (v !== '') {
                    if (!properties[key]) {
                        properties[key] = [];
                    }
                    properties[key].push(v);
                }
                searchValue = searchValue.substr(0, p) + searchValue.substr(i);
            }
            p = searchValue.indexOf(k);
        }
    });

    const values = _.uniq(searchValue.split(' ').filter(v => v.length > 0));

    return { hasKeywords: keywordsCnt > 0, values, properties };
};

const getFirstProp = (sq: ParseSearchQuery, name: string) => sq.properties[name] && sq.properties[name][0];
const getPropValue = (sq: ParseSearchQuery, name: string, value: string) => sq.properties[name] && sq.properties[name].find((v: string) => v === value);
const getProperties = (sq: ParseSearchQuery): PropertyValue[] => {
    if (sq.properties.has) {
        return sq.properties.has.map((value: string) => {
            const v = value.split(':');
            return {
                key: v[0],
                value: v[1]
            };
        });
    }
    return [];
};

export const getAdvancedDataFromQuery = (query: string): SearchBarAdvanceFormData => {
    const sq = parseSearchQuery(query);

    return {
        searchValue: sq.values.join(' '),
        type: getFirstProp(sq, 'type') as ResourceKind,
        cluster: getFirstProp(sq, 'cluster') as ClusterObjectType,
        projectUuid: getFirstProp(sq, 'project'),
        inTrash: getPropValue(sq, 'is', 'trashed') !== undefined,
        dateFrom: getFirstProp(sq, 'from'),
        dateTo: getFirstProp(sq, 'to'),
        properties: getProperties(sq),
        saveQuery: false,
        queryName: ''
    };
};

export const getFilters = (filterName: string, searchValue: string): string => {
    const filter = new FilterBuilder();
    const sq = parseSearchQuery(searchValue);

    const resourceKind = getFirstProp(sq, 'type') as ResourceKind;

    let prefix = '';
    switch (resourceKind) {
        case ResourceKind.COLLECTION:
            prefix = GroupContentsResourcePrefix.COLLECTION;
            break;
        case ResourceKind.PROCESS:
            prefix = GroupContentsResourcePrefix.PROCESS;
            break;
        default:
            prefix = GroupContentsResourcePrefix.PROJECT;
            break;
    }

    if (!sq.hasKeywords) {
        filter
            .addILike(filterName, searchValue, GroupContentsResourcePrefix.COLLECTION)
            .addILike(filterName, searchValue, GroupContentsResourcePrefix.PROCESS)
            .addILike(filterName, searchValue, GroupContentsResourcePrefix.PROJECT);
    } else {
        if (prefix) {
            sq.values.forEach(v =>
                filter.addILike(filterName, v, prefix)
            );
        } else {
            sq.values.forEach(v => {
                filter
                    .addILike(filterName, v, GroupContentsResourcePrefix.COLLECTION)
                    .addILike(filterName, v, GroupContentsResourcePrefix.PROCESS)
                    .addILike(filterName, v, GroupContentsResourcePrefix.PROJECT);
            });
        }

        if (getPropValue(sq, 'is', 'trashed')) {
            filter.addEqual("is_trashed", true);
        }

        const projectUuid = getFirstProp(sq, 'project');
        if (projectUuid) {
            filter.addEqual('uuid', projectUuid, prefix);
        }

        const dateFrom = getFirstProp(sq, 'from');
        if (dateFrom) {
            filter.addGte('modified_at', buildDateFilter(dateFrom));
        }

        const dateTo = getFirstProp(sq, 'to');
        if (dateTo) {
            filter.addLte('modified_at', buildDateFilter(dateTo));
        }

        const props = getProperties(sq);
        props.forEach(p => {
            // filter.addILike(`properties.${p.key}`, p.value);
            filter.addExists(p.key);
        });
    }

    return filter
        .addEqual('groupClass', GroupClass.PROJECT, GroupContentsResourcePrefix.PROJECT)
        .addIsA("uuid", buildUuidFilter(resourceKind))
        .getFilters();
};

const buildUuidFilter = (type?: ResourceKind): ResourceKind[] => {
    return type ? [type] : [ResourceKind.PROJECT, ResourceKind.COLLECTION, ResourceKind.PROCESS];
};

const buildDateFilter = (date?: string): string => {
    return date ? date : '';
};

export const initAdvanceFormProjectsTree = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(initUserProject(SEARCH_BAR_ADVANCE_FORM_PICKER_ID));
    };

export const changeAdvanceFormProperty = (property: string, value: PropertyValue[] | string = '') =>
    (dispatch: Dispatch) => {
        dispatch(change(SEARCH_BAR_ADVANCE_FORM_NAME, property, value));
    };

export const updateAdvanceFormProperties = (propertyValues: PropertyValue) =>
    (dispatch: Dispatch) => {
        dispatch(arrayPush(SEARCH_BAR_ADVANCE_FORM_NAME, 'properties', propertyValues));
    };

export const moveUp = () =>
    (dispatch: Dispatch) => {
        dispatch(searchBarActions.MOVE_UP());
    };

export const moveDown = () =>
    (dispatch: Dispatch) => {
        dispatch(searchBarActions.MOVE_DOWN());
    };
