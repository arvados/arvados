// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { FloatCommandInputParameter, isRequiredInput } from '~/models/workflow';
import { Field } from 'redux-form';
import { isNumber } from '~/validators/is-number';
import { GenericInput } from '~/views/run-process-panel/inputs/generic-input';
import { Input as MaterialInput } from '@material-ui/core';
import { GenericInputProps } from './generic-input';
export interface FloatInputProps {
    input: FloatCommandInputParameter;
}
export const FloatInput = ({ input }: FloatInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={FloatInputComponent}
        parse={parseFloat}
        format={value => isNaN(value) ? '' : JSON.stringify(value)}
        validate={[
            isRequiredInput(input)
                ? isNumber
                : () => undefined,]} />;

class FloatInputComponent extends React.Component<GenericInputProps> {
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
        return <GenericInput
            component={Input}
            {...props} />;
    }
}

const Input = (props: GenericInputProps) =>
    <MaterialInput fullWidth {...props.input} error={props.meta.touched && !!props.meta.error} />;
