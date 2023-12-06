// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TCheckedList } from "components/data-table/data-table";
import { ContainerRequestResource } from "models/container-request";
import { Dispatch } from "redux";
import { navigateTo } from "store/navigation/navigation-action";
import { snackbarActions } from "store/snackbar/snackbar-actions";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { SnackbarKind } from "store/snackbar/snackbar-actions";
import { ContextMenuResource } from 'store/context-menu/context-menu-actions';

export const multiselectActionContants = {
    TOGGLE_VISIBLITY: "TOGGLE_VISIBLITY",
    SET_CHECKEDLIST: "SET_CHECKEDLIST",
    SELECT_ONE: 'SELECT_ONE',
    DESELECT_ONE: "DESELECT_ONE",
    TOGGLE_ONE: 'TOGGLE_ONE',
    SET_SELECTED_UUID: 'SET_SELECTED_UUID',
    ADD_DISABLED: 'ADD_DISABLED',
    REMOVE_DISABLED: 'REMOVE_DISABLED',
};

export const msNavigateToOutput = (resource: ContextMenuResource | ContainerRequestResource) => async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
    try {
        await services.collectionService.get(resource.outputUuid || '');
        dispatch<any>(navigateTo(resource.outputUuid || ''));
    } catch {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Output collection was trashed or deleted.", hideDuration: 4000, kind: SnackbarKind.WARNING }));
    }
};

export const isExactlyOneSelected = (checkedList: TCheckedList) => {
    let tally = 0;
    let current = '';
    for (const uuid in checkedList) {
        if (checkedList[uuid] === true) {
            tally++;
            current = uuid;
        }
    }
    return tally === 1 ? current : null
};

export const toggleMSToolbar = (isVisible: boolean) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.TOGGLE_VISIBLITY, payload: isVisible });
    };
};

export const setCheckedListOnStore = (checkedList: TCheckedList) => {
    return dispatch => {
        dispatch(setSelectedUuid(isExactlyOneSelected(checkedList)))
        dispatch({ type: multiselectActionContants.SET_CHECKEDLIST, payload: checkedList });
    };
};

export const selectOne = (uuid: string) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.SELECT_ONE, payload: uuid });
    };
};

export const deselectOne = (uuid: string) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.DESELECT_ONE, payload: uuid });
    };
};

export const toggleOne = (uuid: string) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.TOGGLE_ONE, payload: uuid });
    };
};

export const setSelectedUuid = (uuid: string | null) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.SET_SELECTED_UUID, payload: uuid });
    };
};

export const addDisabledButton = (buttonName: string) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.ADD_DISABLED, payload: buttonName });
    };
};

export const removeDisabledButton = (buttonName: string) => {
    return dispatch => {
        dispatch({ type: multiselectActionContants.REMOVE_DISABLED, payload: buttonName });
    };
};

export const multiselectActions = {
    toggleMSToolbar,
    setCheckedListOnStore,
    selectOne,
    deselectOne,
    toggleOne,
    setSelectedUuid,
    addDisabledButton,
    removeDisabledButton,
};
