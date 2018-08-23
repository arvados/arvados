// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectTreePickerField } from '~/views-components/project-tree-picker/project-tree-picker';
import { COPY_NAME_VALIDATION, COPY_PROJECT_VALIDATION } from '~/validators/validators';
import { TextField } from "~/components/text-field/text-field";
import { CopyFormDialogData } from "~/store/collection-copy-dialog/collection-copy-dialog";

type CopyFormDialogProps = WithDialogProps<string> & InjectedFormProps<CopyFormDialogData>;

export const CopyFormDialog = (props: CopyFormDialogProps) =>
    <FormDialog
        dialogTitle='Make a copy'
        formFields={CopyFields}
        submitLabel='Copy'
        {...props}
    />;

const CopyFields = () => <span>
    <Field
        name='name'
        component={TextField}
        validate={COPY_NAME_VALIDATION}
        label="Enter a new name for the copy" />
    <Field
        name="ownerUuid"
        component={ProjectTreePickerField}
        validate={COPY_PROJECT_VALIDATION} />
</span>;
