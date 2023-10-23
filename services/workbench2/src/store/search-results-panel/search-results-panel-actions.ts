// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { setBreadcrumbs } from 'store/breadcrumbs/breadcrumbs-actions';
import { searchBarActions } from 'store/search-bar/search-bar-actions';

export const SEARCH_RESULTS_PANEL_ID = "searchResultsPanel";
export const searchResultsPanelActions = bindDataExplorerActions(SEARCH_RESULTS_PANEL_ID);

export const loadSearchResultsPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Search results' }]));
        const loc = getState().router.location;
        if (loc !== null) {
            const search = new URLSearchParams(loc.search);
            const q = search.get('q');
            if (q !== null) {
                dispatch(searchBarActions.SET_SEARCH_VALUE(q));
            }
        }
        dispatch(searchBarActions.SET_SEARCH_RESULTS([]));
        dispatch(searchResultsPanelActions.CLEAR());
        dispatch(searchResultsPanelActions.REQUEST_ITEMS(true));
    };
