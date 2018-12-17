// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { TextField } from '~/components/text-field/text-field';
import { VirtualMachinesResource } from '~/models/virtual-machines';
import { USER_LENGTH_VALIDATION } from '~/validators/validators';
import { InputLabel } from '@material-ui/core';
import { NativeSelectField } from '~/components/select-field/select-field';
import { SETUP_SHELL_ACCOUNT_DIALOG, createUser } from '~/store/users/users-actions';

interface SetupShellAccountFormDialogData {
    email: string;
    virtualMachineName: string;
    groupVirtualMachine: string;
}

export const SetupShellAccountDialog = compose(
    withDialog(SETUP_SHELL_ACCOUNT_DIALOG),
    reduxForm<SetupShellAccountFormDialogData>({
        form: SETUP_SHELL_ACCOUNT_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(createUser(data));
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

const UserEmailField = ({ data }: any) =>
    <Field
        name='email'
        component={TextField}
        disabled
        label={data.email} />;

const UserGroupsVirtualMachineField = () =>
    <Field
        name='groups'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        label="Groups for virtual machine (comma separated list)" />;

type SetupShellAccountDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<SetupShellAccountFormDialogData>;

const SetupShellAccountFormFields = (props: SetupShellAccountDialogComponentProps) =>
    <>
        <UserEmailField data={props.data}/>
        <UserGroupsVirtualMachineField />
    </>;



