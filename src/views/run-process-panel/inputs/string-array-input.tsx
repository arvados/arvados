// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { isRequiredInput, StringArrayCommandInputParameter } from '~/models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from '~/validators/require';
import { GenericInputProps, GenericInput } from '~/views/run-process-panel/inputs/generic-input';
import { ChipsInput } from '../../../components/chips-input/chips-input';

export interface StringArrayInputProps {
    input: StringArrayCommandInputParameter;
}
export const StringArrayInput = ({ input }: StringArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={StringArrayInputComponent}
        validate={[
            isRequiredInput(input)
                ? (value: string[]) => value.length > 0 ? undefined : ERROR_MESSAGE
                : () => undefined,
        ]} />;

const StringArrayInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) =>
    <ChipsInput
        values={props.input.value}
        onChange={props.input.onChange}
        createNewValue={v => v} />;
