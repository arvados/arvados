// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { multiselectActionConstants } from "./multiselect-actions";
import { TCheckedList } from "components/data-table/data-table";

export type MultiselectToolbarState = {
    isVisible: boolean;
    checkedList: TCheckedList;
    disabledButtons: string[];
};

const multiselectToolbarInitialState = {
    isVisible: false,
    checkedList: {},
    disabledButtons: []
};

const uncheckAllOthers = (inputList: TCheckedList, uuid: string) => {
    const checkedlist = {...inputList}
    for (const key in checkedlist) {
        if (key !== uuid) checkedlist[key] = false;
    }
    return checkedlist;
};

const toggleOneCheck = (inputList: TCheckedList, uuid: string)=>{
    const checkedlist = { ...inputList };
    const isOnlyOneSelected = Object.values(checkedlist).filter(x => x === true).length === 1;
    return { ...inputList, [uuid]: (checkedlist[uuid] && checkedlist[uuid] === true) && isOnlyOneSelected ? false : true };
}

const { TOGGLE_VISIBLITY, SET_CHECKEDLIST, SELECT_ONE, DESELECT_ONE, DESELECT_ALL_OTHERS, TOGGLE_ONE, ADD_DISABLED, REMOVE_DISABLED } = multiselectActionConstants;

export const multiselectReducer = (state: MultiselectToolbarState = multiselectToolbarInitialState, action) => {
    switch (action.type) {
        case TOGGLE_VISIBLITY:
            return { ...state, isVisible: action.payload };
        case SET_CHECKEDLIST:
            return { ...state, checkedList: action.payload };
        case SELECT_ONE:
            return { ...state, checkedList: { ...state.checkedList, [action.payload]: true } };
        case DESELECT_ONE:
            return { ...state, checkedList: { ...state.checkedList, [action.payload]: false } };
        case DESELECT_ALL_OTHERS:
            return { ...state, checkedList: uncheckAllOthers(state.checkedList, action.payload) };
        case TOGGLE_ONE:
            return { ...state, checkedList: toggleOneCheck(state.checkedList, action.payload) };
        case ADD_DISABLED:
            return { ...state, disabledButtons: [...state.disabledButtons, action.payload]}
        case REMOVE_DISABLED:
            return { ...state, disabledButtons: state.disabledButtons.filter((button) => button !== action.payload) };
        default:
            return state;
    }
};
