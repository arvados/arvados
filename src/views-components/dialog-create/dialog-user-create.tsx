// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { UserEmailField, UserVirtualMachineField, UserGroupsVirtualMachineField } from 'views-components/form-fields/user-form-fields';

export type DialogUserProps = WithDialogProps<{}> & InjectedFormProps<any>;

export const UserRepositoryCreate = (props: DialogUserProps) =>
    <FormDialog
        dialogTitle='New user'
        formFields={UserAddFields}
        submitLabel='ADD NEW USER'
        {...props}
    />;

const UserAddFields = (props: DialogUserProps) => <span>
    <UserEmailField />
    <UserVirtualMachineField data={props.data}/>
    <UserGroupsVirtualMachineField />
</span>;
