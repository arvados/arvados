// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch, compose } from 'redux';
import { reduxForm, reset, startSubmit, stopSubmit, InjectedFormProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from '~/store/dialog/with-dialog';
import { dialogActions } from "~/store/dialog/dialog-actions";
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { DialogContentText } from '@material-ui/core';
import { TextField } from '~/components/text-field/text-field';

export const RENAME_FILE_DIALOG = 'renameFileDialog';

export const openRenameFileDialog = (originalName: string) =>
    (dispatch: Dispatch) => {
        dispatch(reset(RENAME_FILE_DIALOG));
        dispatch(dialogActions.OPEN_DIALOG({ id: RENAME_FILE_DIALOG, data: originalName }));
    };

export const RenameFileDialog = compose(
    withDialog(RENAME_FILE_DIALOG),
    reduxForm({
        form: RENAME_FILE_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(RENAME_FILE_DIALOG));
            // TODO: call collection file renaming action here
            setTimeout(() => dispatch(stopSubmit(RENAME_FILE_DIALOG, { name: 'Invalid name' })), 2000);
        }
    })
)((props: WithDialogProps<string> & InjectedFormProps<{ name: string }>) =>
    <FormDialog
        dialogTitle='Rename'
        formFields={RenameDialogFormFields}
        submitLabel='Ok'
        {...props}
    />);

const RenameDialogFormFields = (props: WithDialogProps<string>) => <>
    <DialogContentText>
        {`Please, enter a new name for ${props.data}`}
    </DialogContentText>
    <Field
        name='name'
        component={TextField}
    />
</>;
