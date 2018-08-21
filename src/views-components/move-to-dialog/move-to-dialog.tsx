// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field, WrappedFieldProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectTreePicker } from '~/views-components/project-tree-picker/project-tree-picker';
import { Typography } from "@material-ui/core";
import { ResourceKind } from '~/models/resource';
import { MOVE_TO_VALIDATION } from '../../validators/validators';

export interface MoveToFormDialogData {
    name: string;
    uuid: string;
    ownerUuid: string;
    kind: ResourceKind;
}

export const MoveToFormDialog = (props: WithDialogProps<string> & InjectedFormProps<MoveToFormDialogData>) =>
    <FormDialog
        dialogTitle='Move to'
        formFields={MoveToDialogFields}
        submitLabel='Move'
        {...props}
    />;

const MoveToDialogFields = () =>
    <Field
        name="ownerUuid"
        component={ProjectPicker}
        validate={MOVE_TO_VALIDATION} />;

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
