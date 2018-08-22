// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field, WrappedFieldProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectTreePicker } from '~/views-components/project-tree-picker/project-tree-picker';
import { Typography } from "@material-ui/core";
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
        component={ProjectPicker}
        validate={MAKE_A_COPY_VALIDATION} />;

const ProjectPicker = (props: WrappedFieldProps) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectTreePicker onChange={handleChange(props)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) => (value: string) =>
    props.input.value === value
        ? props.input.onChange('')
        : props.input.onChange(value);