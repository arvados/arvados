// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, Field, WrappedFieldArrayProps, FieldArray } from 'redux-form';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CREATE_GROUP_DIALOG, CREATE_GROUP_FORM, createGroup, CreateGroupFormData, CREATE_GROUP_NAME_FIELD_NAME, CREATE_GROUP_USERS_FIELD_NAME } from 'store/groups-panel/groups-panel-actions';
import { TextField } from 'components/text-field/text-field';
import { maxLength } from 'validators/max-length';
import { require } from 'validators/require';
import { ParticipantSelect, Participant } from 'views-components/sharing-dialog/participant-select';

export const CreateGroupDialog = compose(
    withDialog(CREATE_GROUP_DIALOG),
    reduxForm<CreateGroupFormData>({
        form: CREATE_GROUP_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(createGroup(data));
        }
    })
)(
    (props: CreateGroupDialogComponentProps) =>
        <FormDialog
            dialogTitle='Create a group'
            formFields={CreateGroupFormFields}
            submitLabel='Create'
            {...props}
        />
);

type CreateGroupDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<CreateGroupFormData>;

const CreateGroupFormFields = () =>
    <>
        <GroupNameField />
        <UsersField />
    </>;

const GroupNameField = () =>
    <Field
        name={CREATE_GROUP_NAME_FIELD_NAME}
        component={TextField}
        validate={GROUP_NAME_VALIDATION}
        label="Name"
        autoFocus={true} />;

const GROUP_NAME_VALIDATION = [require, maxLength(255)];

const UsersField = () =>
    <FieldArray
        name={CREATE_GROUP_USERS_FIELD_NAME}
        component={UsersSelect} />;

const UsersSelect = ({ fields }: WrappedFieldArrayProps<Participant>) =>
    <ParticipantSelect
        onlyPeople
        label='Enter email adresses '
        items={fields.getAll() || []}
        onSelect={fields.push}
        onDelete={fields.remove} />;
