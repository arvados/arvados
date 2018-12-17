// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps, WrappedFieldArrayProps, FieldArray } from 'redux-form';
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { PeopleSelect, Person } from '~/views-components/sharing-dialog/people-select';
import { ADD_GROUP_MEMBERS_DIALOG, ADD_GROUP_MEMBERS_FORM, AddGroupMembersFormData, ADD_GROUP_MEMBERS_USERS_FIELD_NAME, addGroupMembers } from '~/store/group-details-panel/group-details-panel-actions';
import { minLength } from '~/validators/min-length';

export const AddGroupMembersDialog = compose(
    withDialog(ADD_GROUP_MEMBERS_DIALOG),
    reduxForm<AddGroupMembersFormData>({
        form: ADD_GROUP_MEMBERS_FORM,
        onSubmit: (data, dispatch) => {
            dispatch(addGroupMembers(data));
        },
    })
)(
    (props: AddGroupMembersDialogProps) =>
        <FormDialog
            dialogTitle='Add users'
            formFields={UsersField}
            submitLabel='Add'
            {...props}
        />
);

type AddGroupMembersDialogProps = WithDialogProps<{}> & InjectedFormProps<AddGroupMembersFormData>;

const UsersField = () =>
    <FieldArray
        name={ADD_GROUP_MEMBERS_USERS_FIELD_NAME}
        component={UsersSelect}
        validate={UsersFieldValidation} />;

const UsersFieldValidation = [minLength(1, () => 'Select at least one user')];

const UsersSelect = ({ fields }: WrappedFieldArrayProps<Person>) =>
    <PeopleSelect
        autofocus
        label='Enter email adresses '
        items={fields.getAll() || []}
        onSelect={fields.push}
        onDelete={fields.remove} />;
