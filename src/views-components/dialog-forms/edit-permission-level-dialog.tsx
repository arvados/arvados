// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, WrappedFieldProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { EDIT_PERMISSION_LEVEL_DIALOG, EDIT_PERMISSION_LEVEL_FORM, EditPermissionLevelFormData, EDIT_PERMISSION_LEVEL_FIELD_NAME, editPermissionLevel } from 'store/group-details-panel/group-details-panel-actions';
import { require } from 'validators/require';
import { PermissionSelect } from 'views-components/sharing-dialog/permission-select';
import { Grid } from '@material-ui/core';
import { Resource } from 'models/resource';
import { ResourceLabel } from 'views-components/data-explorer/renderers';

export const EditPermissionLevelDialog = compose(
    withDialog(EDIT_PERMISSION_LEVEL_DIALOG),
    reduxForm<EditPermissionLevelFormData>({
        form: EDIT_PERMISSION_LEVEL_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(editPermissionLevel(data));
        },
    })
)(
    (props: EditPermissionLevelDialogProps) =>
        <FormDialog
            dialogTitle='Edit permission'
            formFields={PermissionField}
            submitLabel='Update'
            {...props}
        />
);

interface EditPermissionLevelDataProps {
    data: Resource;
}

type EditPermissionLevelDialogProps = EditPermissionLevelDataProps & WithDialogProps<{}> & InjectedFormProps<EditPermissionLevelFormData>;

const PermissionField = (props: EditPermissionLevelDialogProps) =>
    <Grid container spacing={8}>
        <Grid item xs={8}>
            <ResourceLabel uuid={props.data.uuid} />
        </Grid>
        <Grid item xs={4} container wrap='nowrap'>
        <Field
            name={EDIT_PERMISSION_LEVEL_FIELD_NAME}
            component={PermissionSelectComponent as any}
            validate={require} />
        </Grid>
    </Grid>;

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <PermissionSelect fullWidth disableUnderline {...input} />;
