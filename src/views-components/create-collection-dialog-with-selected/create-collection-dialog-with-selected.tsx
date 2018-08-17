// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reduxForm, reset, startSubmit, stopSubmit } from "redux-form";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { DialogCollectionCreateWithSelected } from "../dialog-create/dialog-collection-create-selected";

export const DIALOG_COLLECTION_CREATE_WITH_SELECTED = 'dialogCollectionCreateWithSelected';

export const createCollectionWithSelected = () =>
    (dispatch: Dispatch) => {
        dispatch(reset(DIALOG_COLLECTION_CREATE_WITH_SELECTED));
        dispatch(dialogActions.OPEN_DIALOG({ id: DIALOG_COLLECTION_CREATE_WITH_SELECTED, data: {} }));
    };

export const [DialogCollectionCreateWithSelectedFile] = [DialogCollectionCreateWithSelected]
    .map(withDialog(DIALOG_COLLECTION_CREATE_WITH_SELECTED))
    .map(reduxForm({
        form: DIALOG_COLLECTION_CREATE_WITH_SELECTED,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(DIALOG_COLLECTION_CREATE_WITH_SELECTED));
            setTimeout(() => dispatch(stopSubmit(DIALOG_COLLECTION_CREATE_WITH_SELECTED, { name: 'Invalid name' })), 2000);
        }
    }));
