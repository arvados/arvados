// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { RepositoryNameField } from 'views-components/form-fields/repository-form-fields';

type DialogRepositoryProps = WithDialogProps<{}> & InjectedFormProps<any>;

export const DialogRepositoryCreate = (props: DialogRepositoryProps) =>
    <FormDialog
        dialogTitle='Add new repository'
        formFields={RepositoryNameField}
        submitLabel='CREATE REPOSITORY'
        {...props}
    />;


