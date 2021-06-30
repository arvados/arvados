// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps, WrappedFieldInputProps } from 'redux-form';
import { FormGroup, FormLabel, FormHelperText } from '@material-ui/core';

interface FormFieldCustomProps {
    children: <P>(props: WrappedFieldInputProps) => React.ReactElement<P>;
    label?: string;
    helperText?: string;
    required?: boolean;
}

export type FormFieldProps = FormFieldCustomProps & WrappedFieldProps;

export const FormField = ({ children, ...props }: FormFieldProps & WrappedFieldProps) => {
    return (
        <FormGroup>

            <FormLabel
                focused={props.meta.active}
                required={props.required}
                error={props.meta.touched && !!props.meta.error}>
                {props.label}
            </FormLabel>

            { children(props.input) }

            <FormHelperText error={props.meta.touched && !!props.meta.error}>
                {
                    props.meta.touched && props.meta.error
                        ? props.meta.error
                        : props.helperText
                }
            </FormHelperText>

        </FormGroup>
    );
};
