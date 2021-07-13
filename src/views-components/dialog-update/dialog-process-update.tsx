// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProcessUpdateFormDialogData } from 'store/processes/process-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProcessNameField, ProcessDescriptionField } from 'views-components/form-fields/process-form-fields';

type DialogProcessProps = WithDialogProps<{}> & InjectedFormProps<ProcessUpdateFormDialogData>;

export const DialogProcessUpdate = (props: DialogProcessProps) =>
    <FormDialog
        dialogTitle='Edit Process'
        formFields={ProcessEditFields}
        submitLabel='Save'
        {...props}
    />;

const ProcessEditFields = () => <span>
    <ProcessNameField />
    <ProcessDescriptionField />
</span>;
