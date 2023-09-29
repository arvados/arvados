// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, Field, GenericField } from 'redux-form';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { VIRTUAL_MACHINE_ADD_LOGIN_DIALOG, VIRTUAL_MACHINE_ADD_LOGIN_FORM, addUpdateVirtualMachineLogin, AddLoginFormData, VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD, VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD } from 'store/virtual-machines/virtual-machines-actions';
import { ParticipantSelect } from 'views-components/sharing-dialog/participant-select';
import { GroupArrayInput, GroupArrayDataProps } from 'views-components/virtual-machines-dialog/group-array-input';

export const VirtualMachineAddLoginDialog = compose(
    withDialog(VIRTUAL_MACHINE_ADD_LOGIN_DIALOG),
    reduxForm<AddLoginFormData>({
        form: VIRTUAL_MACHINE_ADD_LOGIN_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(addUpdateVirtualMachineLogin(data));
        }
    })
)(
    (props: CreateGroupDialogComponentProps) => {
        const [hasPartialGroupInput, setPartialGroupInput] = React.useState<boolean>(false);

        return <FormDialog
            dialogTitle={props.data.updating ? "Update login permission" : "Add login permission"}
            formFields={AddLoginFormFields}
            submitLabel={props.data.updating ? "Update" : "Add"}
            {...props}
            data={{
                ...props.data,
                setPartialGroupInput,
                hasPartialGroupInput,
            }}
            invalid={props.invalid || hasPartialGroupInput}
        />;
    }
);

type CreateGroupDialogComponentProps = WithDialogProps<{updating: boolean}> & GroupArrayDataProps & InjectedFormProps<AddLoginFormData>;

const AddLoginFormFields = (props) => {
    return <>
        <ParticipantField
            name={VIRTUAL_MACHINE_ADD_LOGIN_USER_FIELD}
            component={props.data.updating ? ReadOnlyUserSelect : UserSelect}
            excludedParticipants={props.data.excludedParticipants}
        />
        <GroupArrayInput
            name={VIRTUAL_MACHINE_ADD_LOGIN_GROUPS_FIELD}
            input={{id:"Add groups to VM login (eg: docker, sudo)", disabled:false}}
            required={false}
            setPartialGroupInput={props.data.setPartialGroupInput}
            hasPartialGroupInput={props.data.hasPartialGroupInput}
        />
    </>;
}

interface UserFieldProps {
    excludedParticipants: string[];
}

const ParticipantField = Field as new () => GenericField<UserFieldProps>;

const UserSelect = (props) =>
    <ParticipantSelect
        onlyPeople
        onlyActive
        label='Search for user to grant login permission'
        items={props.input.value ? [props.input.value] : []}
        excludedParticipants={props.excludedParticipants}
        onSelect={props.input.onChange}
        onDelete={() => (props.input.onChange(''))} />;

const ReadOnlyUserSelect = (props) =>
        <ParticipantSelect
            onlyPeople
            onlyActive
            label='User'
            items={props.input.value ? [props.input.value] : []}
            disabled={true} />;
