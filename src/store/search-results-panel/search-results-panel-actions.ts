// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';

export const SEARCH_RESULTS_PANEL_ID = "searchResultsPanel";
export const searchResultsPanelActions = bindDataExplorerActions(SEARCH_RESULTS_PANEL_ID);

export const loadSearchResultsPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Search results' }]));
        dispatch(searchResultsPanelActions.REQUEST_ITEMS());
    };