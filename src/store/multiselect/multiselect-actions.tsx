// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export const multiselectActions = {
    TOGGLE_VISIBLITY: 'TOGGLE_VISIBLITY',
    SET_CHECKEDLIST: 'SET_CHECKEDLIST',
};

export const toggleMSToolbar = (isVisible: boolean) => {
    return (dispatch) => {
        dispatch({ type: multiselectActions.TOGGLE_VISIBLITY, payload: isVisible });
    };
};

export const setCheckedListOnStore = (checkedList) => {
    return (dispatch) => {
        dispatch({ type: multiselectActions.SET_CHECKEDLIST, payload: checkedList });
    };
};
