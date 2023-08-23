// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from 'store/dialog/dialog-actions';

export const NOT_FOUND_DIALOG_NAME = 'notFoundDialog';

export const openNotFoundDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: NOT_FOUND_DIALOG_NAME,
            data: {},
        }));
    };