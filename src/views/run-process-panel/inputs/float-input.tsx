// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { getInputLabel, FloatCommandInputParameter, isRequiredInput } from '~/models/workflow';
import { Field, WrappedFieldProps } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { isNumber } from '~/validators/is-number';
export interface FloatInputProps {
    input: FloatCommandInputParameter;
}
export const FloatInput = ({ input }: FloatInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={DecimalInput}
        parse={parseFloat}
        format={value => isNaN(value) ? '' : JSON.stringify(value)}
        validate={[
            isRequiredInput(input)
                ? isNumber
                : () => undefined,]} />;

class DecimalInput extends React.Component<WrappedFieldProps & { label?: string }> {
    state = {
        endsWithDecimalSeparator: false,
    };

    handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const [base, fraction] = event.target.value.split('.');
        this.setState({ endsWithDecimalSeparator: fraction === '' });
        this.props.input.onChange(event);
    }

    render() {
        const props = {
            ...this.props,
            input: {
                ...this.props.input,
                value: this.props.input.value + (this.state.endsWithDecimalSeparator ? '.' : ''),
                onChange: this.handleChange,
            },
        };
        return <TextField {...props} />;
    }
}
