// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';

export const searchBarActions = unionize({
    SET_CURRENT_VIEW: ofType<string>(),
    OPEN_SEARCH_VIEW: ofType<{}>(),
    CLOSE_SEARCH_VIEW: ofType<{}>()
});

export type SearchBarActions = UnionOf<typeof searchBarActions>;

export const goToView = (currentView: string) => searchBarActions.SET_CURRENT_VIEW(currentView);

export const saveRecentQuery = (query: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        services.searchQueriesService.saveRecentQuery(query);
    };

export const loadRecentQueries = () =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const recentSearchQueries = services.searchQueriesService.getRecentQueries();
        return recentSearchQueries || [];
    };