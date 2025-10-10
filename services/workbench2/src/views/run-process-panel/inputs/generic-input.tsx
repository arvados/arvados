// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { FormHelperText, InputLabel, FormControl } from '@mui/material';
import { GenericCommandInputParameter, getInputLabel, isRequiredInput } from 'models/workflow';

export type GenericInputProps = WrappedFieldProps & {
    commandInput: GenericCommandInputParameter<any, any>;
};

type GenericInputContainerProps = GenericInputProps & {
    component: React.ComponentType<GenericInputProps>;
    required?: boolean;
};
export const GenericInput = ({ component: Component, ...props }: GenericInputContainerProps) => {
    return <FormControl fullWidth>
        <InputLabel
            shrink
            variant={"standard"} // Filled and outlined cause a left gap
            focused={props.meta.active}
            required={props.required !== undefined ? props.required : isRequiredInput(props.commandInput)}
            error={props.meta.touched && !!props.meta.error}>
            {getInputLabel(props.commandInput)}
        </InputLabel>
        <Component {...props} />
        <FormHelperText error={props.meta.touched && !!props.meta.error}>
            {
                props.meta.touched && props.meta.error
                    ? props.meta.error
                    : props.commandInput.doc
            }
        </FormHelperText>
    </FormControl>;
};
