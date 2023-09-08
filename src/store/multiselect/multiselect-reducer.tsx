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

export const multiselectReducer = (state: MultiselectToolbarState = multiselectToolbarInitialState, action) => {
    if (action.type === multiselectActions.TOGGLE_VISIBLITY) return { ...state, isVisible: action.payload };
    if (action.type === multiselectActions.SET_CHECKEDLIST) return { ...state, checkedList: action.payload };
    if (action.type === multiselectActions.DESELECT_ONE) {
        return { ...state, checkedList: { ...state.checkedList, [action.payload]: false } };
    }
    return state;
};

const updateCheckedList = (uuid: string, newValue: boolean, checkedList: TCheckedList) => {
    return;
    // const newList = { ...checkedList };
    // newList[uuid] = newValue;
    // return newList;
};
