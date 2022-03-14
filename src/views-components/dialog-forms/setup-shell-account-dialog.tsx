// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { TextField } from 'components/text-field/text-field';
import { VirtualMachinesResource } from 'models/virtual-machines';
import { InputLabel } from '@material-ui/core';
import { SETUP_SHELL_ACCOUNT_DIALOG, setupUserVM } from 'store/users/users-actions';
import { UserResource } from 'models/user';
import { VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD, AddLoginFormData } from 'store/virtual-machines/virtual-machines-actions';
import { UserGroupsVirtualMachineField, RequiredUserVirtualMachineField } from 'views-components/form-fields/user-form-fields';

export const SetupShellAccountDialog = compose(
    withDialog(SETUP_SHELL_ACCOUNT_DIALOG),
    reduxForm<AddLoginFormData>({
        form: SETUP_SHELL_ACCOUNT_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(setupUserVM(data));
        }
    })
)(
    (props: SetupShellAccountDialogComponentProps) =>
        <FormDialog
            dialogTitle='Setup shell account'
            formFields={SetupShellAccountFormFields}
            submitLabel='Submit'
            {...props}
        />
);

interface DataProps {
    user: UserResource;
    items: VirtualMachinesResource[];
}

const UserNameField = () =>
    <span>
        <InputLabel>VM Login</InputLabel>
        <Field
            name={`${VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD}.username`}
            component={TextField as any}
            disabled />
    </span>;

type SetupShellAccountDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<AddLoginFormData>;

const SetupShellAccountFormFields = (props: SetupShellAccountDialogComponentProps) =>
    <>
        <UserNameField />
        <RequiredUserVirtualMachineField data={props.data as DataProps} />
        <UserGroupsVirtualMachineField />
    </>;
