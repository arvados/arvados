// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const multiselectActions = {
    TOGGLE_VISIBLITY: 'TOGGLE_VISIBLITY',
};

export const toggleMSToolbar = (isVisible: boolean) => {
    return (dispatch) => {
        dispatch({ type: multiselectActions.TOGGLE_VISIBLITY, payload: isVisible });
    };
};
