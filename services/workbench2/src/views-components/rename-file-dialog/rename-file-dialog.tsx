// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose, Dispatch } from 'redux';
import { reduxForm, InjectedFormProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { DialogContentText } from '@material-ui/core';
import { TextField } from 'components/text-field/text-field';
import { RENAME_FILE_DIALOG, RenameFileDialogData, renameFile } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { WarningCollection } from 'components/warning-collection/warning-collection';
import { RENAME_FILE_VALIDATION } from 'validators/validators';

export const RenameFileDialog = compose(
    withDialog(RENAME_FILE_DIALOG),
    reduxForm({
        form: RENAME_FILE_DIALOG,
        touchOnChange: true,
        onSubmit: (data: { path: string }, dispatch: Dispatch) => {
            dispatch<any>(renameFile(data.path));
        }
    })
)((props: WithDialogProps<RenameFileDialogData> & InjectedFormProps<{ name: string, path: string }>) =>
    <FormDialog
        dialogTitle='Rename'
        formFields={RenameDialogFormFields}
        submitLabel='Ok'
        {...props}
    />);

const RenameDialogFormFields = (props: WithDialogProps<RenameFileDialogData>) => <>
    <DialogContentText>
        {`Please, enter a new name for ${props.data.name}`}
    </DialogContentText>
    <Field
        name='path'
        component={TextField as any}
        autoFocus={true}
        validate={RENAME_FILE_VALIDATION}
    />
    <WarningCollection text="Renaming a file will change the collection's content address." />
</>;
