// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { WorkflowTreePickerField } from '~/views-components/workflow-tree-picker/workflow-tree-picker';
import { MOVE_TO_VALIDATION } from '~/validators/validators';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';

export const DialogMoveTo = (props: WithDialogProps<string> & InjectedFormProps<MoveToFormDialogData>) =>
    <FormDialog
        dialogTitle='Move to'
        formFields={MoveToDialogFields}
        submitLabel='Move'
        {...props}
    />;

const MoveToDialogFields = () =>
    <Field
        name="ownerUuid"
        component={WorkflowTreePickerField}
        validate={MOVE_TO_VALIDATION} />;

