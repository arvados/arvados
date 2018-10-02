// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    getInputLabel,
    isRequiredInput,
    FileCommandInputParameter,
    File,
    CWLType
} from '~/models/workflow';
import { Field } from 'redux-form';
import { require } from '~/validators/require';
import { Input } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';

export interface FileInputProps {
    input: FileCommandInputParameter;
}
export const FileInput = ({ input }: FileInputProps) =>
    <Field
        name={input.id}
        commandInput={input}        
        component={FileInputComponent}
        format={(value?: File) => value ? value.location : ''}
        parse={(value: string): File => ({
            class: CWLType.FILE,
            location: value,
            basename: value.split('/').slice(1).join('/')
        })}
        validate={[
            isRequiredInput(input)
                ? require
                : () => undefined,
        ]} />;

const FileInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={props =>
            <Input readOnly fullWidth value={props.input.value} error={props.meta.touched && !!props.meta.error}/>}
        {...props} />;
