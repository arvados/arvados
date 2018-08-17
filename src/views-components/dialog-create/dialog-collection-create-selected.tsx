// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field, WrappedFieldProps } from "redux-form";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { TextField } from "~/components/text-field/text-field";
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION, COLLECTION_PROJECT_VALIDATION } from "~/validators/validators";
import { ProjectTreePicker } from "../project-tree-picker/project-tree-picker";
import { FormDialog } from '../../components/form-dialog/form-dialog';

export const DialogCollectionCreateWithSelected = (props: WithDialogProps<string> & InjectedFormProps<{ name: string }>) =>
    <FormDialog
        dialogTitle='Create a collection'
        formFields={FormFields}
        submitLabel='Create a collection'
        {...props}
    />;

const FormFields = () => <div style={{ display: 'flex' }}>
    <div>
        <Field
            name='name'
            component={TextField}
            validate={COLLECTION_NAME_VALIDATION}
            label="Collection Name" />
        <Field
            name='description'
            component={TextField}
            validate={COLLECTION_DESCRIPTION_VALIDATION}
            label="Description - optional" />
    </div>
    <Field
        name="projectUuid"
        component={Picker}
        validate={COLLECTION_PROJECT_VALIDATION} />
</div>;

const Picker = (props: WrappedFieldProps) =>
    <div style={{ width: '400px', height: '144px', display: 'flex', flexDirection: 'column' }}>
        <ProjectTreePicker onChange={projectUuid => props.input.onChange(projectUuid)} />
    </div>;
