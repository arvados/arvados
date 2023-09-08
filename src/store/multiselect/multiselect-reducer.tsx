// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { multiselectActions } from "./multiselect-actions";
import { TCheckedList } from "components/data-table/data-table";

type MultiselectToolbarState = {
    isVisible: boolean;
    checkedList: TCheckedList;
};

const multiselectToolbarInitialState = {
    isVisible: false,
    checkedList: {},
};

const { TOGGLE_VISIBLITY, SET_CHECKEDLIST, DESELECT_ONE } = multiselectActions;

export const multiselectReducer = (state: MultiselectToolbarState = multiselectToolbarInitialState, action) => {
    switch (action.type) {
        case TOGGLE_VISIBLITY:
            return { ...state, isVisible: action.payload };
        case SET_CHECKEDLIST:
            return { ...state, checkedList: action.payload };
        case DESELECT_ONE:
            return { ...state, checkedList: { ...state.checkedList, [action.payload]: false } };
        default:
            return state;
    }
};
