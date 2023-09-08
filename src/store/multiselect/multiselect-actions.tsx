// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TCheckedList } from "components/data-table/data-table";

export const multiselectActions = {
    TOGGLE_VISIBLITY: "TOGGLE_VISIBLITY",
    SET_CHECKEDLIST: "SET_CHECKEDLIST",
    DESELECT_ONE: "DESELECT_ONE",
};

export const toggleMSToolbar = (isVisible: boolean) => {
    return dispatch => {
        dispatch({ type: multiselectActions.TOGGLE_VISIBLITY, payload: isVisible });
    };
};

export const setCheckedListOnStore = (checkedList: TCheckedList) => {
    return dispatch => {
        dispatch({ type: multiselectActions.SET_CHECKEDLIST, payload: checkedList });
    };
};

export const deselectOne = (uuid: string) => {
    return dispatch => {
        dispatch({ type: multiselectActions.DESELECT_ONE, payload: uuid });
    };
};
