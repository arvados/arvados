// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { structuredSearchActions, StructuredSearchActions } from '~/store/structured-search/structured-search-actions';

interface StructuredSearch {
    currentView: string;
}

export enum SearchView {
    BASIC = 'basic',
    ADVANCED = 'advanced',
    AUTOCOMPLETE = 'autocomplete'
}

const initialState: StructuredSearch = {
    currentView: SearchView.BASIC
};

export const structuredSearchReducer = (state = initialState, action: StructuredSearchActions): StructuredSearch => 
    structuredSearchActions.match(action, {
        SET_CURRENT_VIEW: currentView => ({... state, currentView}),
        default: () => state
    });