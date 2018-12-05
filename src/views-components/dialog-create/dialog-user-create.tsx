// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { UserFirstNameField, UserLastNameField, UserEmailField, UserIdentityUrlField, UserVirtualMachineField, UserGroupsVirtualMachineField } from '~/views-components/form-fields/user-form-fields';

type DialogUserProps = WithDialogProps<{}> & InjectedFormProps<any>;

export const UserRepositoryCreate = (props: DialogUserProps) =>
    <FormDialog
        dialogTitle='New user'
        formFields={UserAddFields}
        submitLabel='ADD NEW USER'
        {...props}
    />;

const UserAddFields = () => <span>
    <UserFirstNameField />
    <UserLastNameField />
    <UserEmailField />
    <UserIdentityUrlField />
    <UserVirtualMachineField />
    <UserGroupsVirtualMachineField />
</span>;
