// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';

export const sidePanelActions = {
    TOGGLE_COLLAPSE: 'TOGGLE_COLLAPSE'
}

export const navigateFromSidePanel = (id: string) =>
    (dispatch: Dispatch) => {
        dispatch<any>(navigateTo(id));
    };

export const toggleSidePanel = (collapsedState: boolean) => {
    return (dispatch) => {
        dispatch({type: sidePanelActions.TOGGLE_COLLAPSE, payload: !collapsedState})
    }
}