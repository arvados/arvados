// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { isRequiredInput, StringArrayCommandInputParameter } from '~/models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from '~/validators/require';
import { GenericInputProps, GenericInput } from '~/views/run-process-panel/inputs/generic-input';
import { ChipsInput } from '~/components/chips-input/chips-input';
import { identity } from 'lodash';
import { createSelector } from 'reselect';

export interface StringArrayInputProps {
    input: StringArrayCommandInputParameter;
}
export const StringArrayInput = ({ input }: StringArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={StringArrayInputComponent}
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

const StringArrayInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

class Input extends React.PureComponent<GenericInputProps>{
    render() {
        return <ChipsInput
            values={this.props.input.value}
            onChange={this.handleChange}
            createNewValue={identity} />;
    }

    handleChange = (values: {}[]) => {
        const { input, meta } = this.props;
        if (!meta.touched) {
            input.onBlur(values);
        }
        input.onChange(values);
    }
}
