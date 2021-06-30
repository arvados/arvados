// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { isRequiredInput, FloatArrayCommandInputParameter } from 'models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from 'validators/require';
import { GenericInputProps, GenericInput } from 'views/run-process-panel/inputs/generic-input';
import { ChipsInput } from 'components/chips-input/chips-input';
import { createSelector } from 'reselect';
import { FloatInput } from 'components/float-input/float-input';

export interface FloatArrayInputProps {
    input: FloatArrayCommandInputParameter;
}
export const FloatArrayInput = ({ input }: FloatArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={FloatArrayInputComponent}
        validate={validationSelector(input)} />;


const validationSelector = createSelector(
    isRequiredInput,
    isRequired => isRequired
        ? [required]
        : undefined
);

const required = (value: string[]) =>
    value.length > 0
        ? undefined
        : ERROR_MESSAGE;

const FloatArrayInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={InputComponent}
        {...props} />;

class InputComponent extends React.PureComponent<GenericInputProps>{
    render() {
        const { commandInput, input, meta } = this.props;
        return <ChipsInput
            deletable={!commandInput.disabled}
            orderable={!commandInput.disabled}
            disabled={commandInput.disabled}
            values={input.value}
            onChange={this.handleChange}
            createNewValue={parseFloat}
            inputComponent={FloatInput}
            inputProps={{
                error: meta.error,
            }} />;
    }

    handleChange = (values: {}[]) => {
        const { input, meta } = this.props;
        if (!meta.touched) {
            input.onBlur(values);
        }
        input.onChange(values);
    }
}
