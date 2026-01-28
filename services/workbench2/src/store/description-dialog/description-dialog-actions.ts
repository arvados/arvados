// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { RootState } from "store/store";

export const DESCRIPTION_DIALOG = 'DESCRIPTION_DIALOG';

export type DescriptionDialogData = { uuid: string };

export const openDialog = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({id: DESCRIPTION_DIALOG, data: { uuid }}));
    };

export const closeDialog = () =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.CLOSE_DIALOG({id: DESCRIPTION_DIALOG}));
    };

const descriptionDialogActions = {
    openDialog,
    closeDialog
};

export default descriptionDialogActions;
