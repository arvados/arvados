// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, WrappedFieldProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { VIRTUAL_MACHINE_ADD_LOGIN_DIALOG, VIRTUAL_MACHINE_ADD_LOGIN_FORM, addVirtualMachineLogin, AddLoginFormData, VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD } from 'store/virtual-machines/virtual-machines-actions';
import { ParticipantSelect } from 'views-components/sharing-dialog/participant-select';
import { GroupArrayInput } from 'views-components/virtual-machines-dialog/group-array-input';

export const VirtualMachineAddLoginDialog = compose(
    withDialog(VIRTUAL_MACHINE_ADD_LOGIN_DIALOG),
    reduxForm<AddLoginFormData>({
        form: VIRTUAL_MACHINE_ADD_LOGIN_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(addVirtualMachineLogin(data));
        }
    })
)(
    (props: CreateGroupDialogComponentProps) =>
        <FormDialog
            dialogTitle='Add login permissions'
            formFields={AddLoginFormFields}
            submitLabel='Add'
            {...props}
        />
);

type CreateGroupDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<AddLoginFormData>;

const AddLoginFormFields = () =>
    <>
        <UserField />
        <GroupArrayInput
            name={VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD}
            input={{id:"Add groups to VM login", disabled:false}}
            required={false}
        />
    </>;

const UserField = () =>
    <Field
        name={VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD}
        component={UserSelect}
        />;

const UserSelect = ({ input, meta }: WrappedFieldProps) =>
    <ParticipantSelect
        onlyPeople
        label='Search for users to grant login permission'
        items={input.value ? [input.value] : []}
        onSelect={input.onChange}
        onDelete={() => (input.onChange(''))} />;
