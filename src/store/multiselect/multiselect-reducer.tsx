// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { multiselectActions } from './multiselect-actions';

type MultiselectToolbarState = {
    isVisible: boolean;
};

const multiselectToolbarInitialState = {
    isVisible: false,
};

export const multiselectReducer = (state: MultiselectToolbarState = multiselectToolbarInitialState, action) => {
    if (action.type === multiselectActions.TOGGLE_VISIBLITY) return { ...state, isVisible: action.payload };
    return state;
};
