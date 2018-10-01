// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { getInputLabel, isRequiredInput, FileCommandInputParameter, File } from '~/models/workflow';
import { Field } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { require } from '~/validators/require';

export interface FileInputProps {
    input: FileCommandInputParameter;
}
export const FileInput = ({ input }: FileInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={TextField}
        format={(value?: File) => value ? value.location : ''}
        validate={[
            isRequiredInput(input)
                ? require
                : () => undefined,
        ]} />;

