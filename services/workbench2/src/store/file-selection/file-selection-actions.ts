// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { resetPickerProjectTree } from 'store/project-tree-picker/project-tree-picker-actions';

export const FILE_SELECTION = 'fileSelection';

export const openFileSelectionDialog = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        dispatch(dialogActions.OPEN_DIALOG({ id: FILE_SELECTION, data: {} }));
    };