// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from "redux";
import { reduxForm, reset, startSubmit, stopSubmit } from "redux-form";
import { withDialog } from "~/store/dialog/with-dialog";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { loadProjectTreePickerProjects } from "../project-tree-picker/project-tree-picker";
import { CollectionPartialCopyFormDialog } from "~/views-components/form-dialog/collection-form-dialog";

export const COLLECTION_PARTIAL_COPY = 'COLLECTION_PARTIAL_COPY';

export const openCollectionPartialCopyDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(reset(COLLECTION_PARTIAL_COPY));
        dispatch<any>(loadProjectTreePickerProjects(''));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_PARTIAL_COPY, data: {} }));
    };


export const CollectionPartialCopyDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY),
    reduxForm({
        form: COLLECTION_PARTIAL_COPY,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(COLLECTION_PARTIAL_COPY));
            setTimeout(() => dispatch(stopSubmit(COLLECTION_PARTIAL_COPY, { name: 'Invalid name' })), 2000);
        }
    }))(CollectionPartialCopyFormDialog);
