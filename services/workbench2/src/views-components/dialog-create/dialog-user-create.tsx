// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { UserEmailField, UserVirtualMachineField, UserGroupsVirtualMachineField } from 'views-components/form-fields/user-form-fields';
import { UserCreateFormDialogData } from 'store/users/users-actions';
import { UserResource } from 'models/user';
import { VirtualMachinesResource } from 'models/virtual-machines';

export type DialogUserProps = WithDialogProps<{}> & InjectedFormProps<UserCreateFormDialogData>;

interface DataProps {
    user: UserResource;
    items: VirtualMachinesResource[];
}

export const UserRepositoryCreate = (props: DialogUserProps) =>
    <FormDialog
        dialogTitle='New user'
        formFields={UserAddFields}
        submitLabel='ADD NEW USER'
        {...props}
    />;

const UserAddFields = (props: DialogUserProps) => <span>
    <UserEmailField />
    <UserVirtualMachineField data={props.data as DataProps}/>
    <UserGroupsVirtualMachineField />
</span>;
