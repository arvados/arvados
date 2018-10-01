// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { require } from '~/validators/require';
import { WorkflowTreePickerField } from '~/views-components/workflow-tree-picker/workflow-tree-picker';

type FileSelectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogFileSelection = (props: FileSelectionProps) =>
    <FormDialog
        dialogTitle='Choose a file'
        formFields={FileSelectionFields}
        submitLabel='Ok'
        {...props}
    />;

const FileSelectionFields = () =>
    <Field
        name='tree'
        validate={FILES_FIELD_VALIDATION}
        component={WorkflowTreePickerField} />;

const FILES_FIELD_VALIDATION = [require];