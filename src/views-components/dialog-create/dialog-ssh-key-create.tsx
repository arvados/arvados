// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { SshKeyPublicField, SshKeyNameField } from 'views-components/form-fields/ssh-key-form-fields';
import { SshKeyCreateFormDialogData } from 'store/auth/auth-action-ssh';

type DialogSshKeyProps = WithDialogProps<{}> & InjectedFormProps<SshKeyCreateFormDialogData>;

export const DialogSshKeyCreate = (props: DialogSshKeyProps) =>
    <FormDialog
        dialogTitle='Add new SSH key'
        formFields={SshKeyAddFields}
        submitLabel='Add new ssh key'
        {...props}
    />;

const SshKeyAddFields = () => <span>
    <SshKeyPublicField />
    <SshKeyNameField />
</span>;
