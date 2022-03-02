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
import { CHOOSE_VM_VALIDATION } from 'validators/validators';
import { InputLabel } from '@material-ui/core';
import { NativeSelectField } from 'components/select-field/select-field';
import { SETUP_SHELL_ACCOUNT_DIALOG, setupUserVM } from 'store/users/users-actions';
import { UserResource } from 'models/user';
import { VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD, AddLoginFormData } from 'store/virtual-machines/virtual-machines-actions';
import { GroupArrayInput } from 'views-components/virtual-machines-dialog/group-array-input';

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

interface VirtualMachinesProps {
    data: {
        items: VirtualMachinesResource[];
    };
}
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
            disabled /></span>;

const UserVirtualMachineField = ({ data }: VirtualMachinesProps) =>
    <div style={{ marginBottom: '21px' }}>
        <InputLabel>Virtual Machine</InputLabel>
        <Field
            name={VIRTUAL_MACHINE_ADD_LOGIN_VM_FIELD}
            component={NativeSelectField as any}
            validate={CHOOSE_VM_VALIDATION}
            items={getVirtualMachinesList(data.items)} />
    </div>;

const UserGroupsVirtualMachineField = () =>
    <GroupArrayInput
        name={VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD}
        input={{id:"Add groups to VM login (eg: docker, sudo)", disabled:false}}
        required={false}
    />

const getVirtualMachinesList = (virtualMachines: VirtualMachinesResource[]) =>
    [{ key: "", value: "" }].concat(virtualMachines.map(it => ({ key: it.uuid, value: it.hostname })));

type SetupShellAccountDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<AddLoginFormData>;

const SetupShellAccountFormFields = (props: SetupShellAccountDialogComponentProps) =>
    <>
        <UserNameField />
        <UserVirtualMachineField data={props.data as DataProps} />
        <UserGroupsVirtualMachineField />
    </>;
