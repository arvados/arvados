// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectTreePickerField } from '~/views-components/project-tree-picker/project-tree-picker';
import { COPY_NAME_VALIDATION, MAKE_A_COPY_VALIDATION } from '~/validators/validators';
import { TextField } from "~/components/text-field/text-field";
import { ProjectCopyFormDialogData } from "~/store/project-copy-dialog/project-copy-dialog";

export const ProjectCopyFormDialog = (props: WithDialogProps<string> & InjectedFormProps<ProjectCopyFormDialogData>) =>
    <FormDialog
        dialogTitle='Make a copy'
        formFields={ProjectCopyFields}
        submitLabel='Copy'
        {...props}
    />;

const ProjectCopyFields = () => <div>
    <ProjectCopyNameField />
    <ProjectCopyDialogFields />
</div>;

const ProjectCopyNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={COPY_NAME_VALIDATION}
        label="Enter a new name for the copy" />;

const ProjectCopyDialogFields = () =>
    <Field
        name="projectUuid"
        component={ProjectTreePickerField}
        validate={MAKE_A_COPY_VALIDATION} />;
