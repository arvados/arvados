// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from 'lodash/fp';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectTreePickerField } from 'views-components/projects-tree-picker/tree-picker-field';
import { COPY_NAME_VALIDATION, COPY_FILE_VALIDATION } from 'validators/validators';
import { TextField } from "components/text-field/text-field";
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { PickerIdProp } from 'store/tree-picker/picker-id';

type CopyFormDialogProps = WithDialogProps<string> & InjectedFormProps<CopyFormDialogData>;

export const DialogCopy = (props: CopyFormDialogProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Make a copy'
        formFields={CopyDialogFields(props.pickerId)}
        submitLabel='Copy'
        {...props}
    />;

const CopyDialogFields = memoize((pickerId: string) =>
    () =>
        <span>
            <Field
                name='name'
                component={TextField}
                validate={COPY_NAME_VALIDATION}
                label="Enter a new name for the copy" />
            <Field
                name="ownerUuid"
                component={ProjectTreePickerField}
                validate={COPY_FILE_VALIDATION}
                pickerId={pickerId}/>
        </span>);
