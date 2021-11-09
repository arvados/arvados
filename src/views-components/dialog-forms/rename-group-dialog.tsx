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
import { GroupResource } from 'models/group';
import { RENAME_GROUP_DIALOG, RENAME_GROUP_NAME_FIELD_NAME, RenameGroupFormData, renameGroup } from 'store/groups-panel/groups-panel-actions';
// import { WarningCollection } from 'components/warning-collection/warning-collection';
import { RENAME_FILE_VALIDATION } from 'validators/validators';

export const RenameGroupDialog = compose(
    withDialog(RENAME_GROUP_DIALOG),
    reduxForm<RenameGroupFormData>({
        form: RENAME_GROUP_DIALOG,
        // touchOnChange: true,
        onSubmit: (data: RenameGroupFormData, dispatch: Dispatch) => {
            console.log(data);
            // dispatch<any>(renameGroup(data));
        }
    })
)((props: RenameGroupDialogProps) =>
    <FormDialog
        dialogTitle='Rename'
        formFields={RenameGroupFormFields}
        submitLabel='Ok'
        {...props}
    />);

interface RenameGroupDataProps {
    data: GroupResource;
}

type RenameGroupDialogProps = RenameGroupDataProps & WithDialogProps<{}> & InjectedFormProps<RenameGroupFormData>;

const RenameGroupFormFields = (props: RenameGroupDialogProps) => {
    // console.log(props);
    return <>
        <DialogContentText>
            {`Please enter a new name for ${props.data.name}`}
        </DialogContentText>
        <Field
            name={RENAME_GROUP_NAME_FIELD_NAME}
            component={TextField as any}
            autoFocus={true}
            validate={RENAME_FILE_VALIDATION}
        />
        {/* <WarningCollection text="Renaming a file will change the collection's content address." /> */}
    </>;
}
