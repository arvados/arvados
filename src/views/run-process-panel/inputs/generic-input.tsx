// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { FormGroup, FormLabel, FormHelperText } from '@material-ui/core';
import { GenericCommandInputParameter, getInputLabel, isRequiredInput } from 'models/workflow';

export type GenericInputProps = WrappedFieldProps & {
    commandInput: GenericCommandInputParameter<any, any>;
};

type GenericInputContainerProps = GenericInputProps & {
    component: React.ComponentType<GenericInputProps>;
};
export const GenericInput = ({ component: Component, ...props }: GenericInputContainerProps) => {
    return <FormGroup>
        <FormLabel
            focused={props.meta.active}
            required={isRequiredInput(props.commandInput)}
            error={props.meta.touched && !!props.meta.error}>
            {getInputLabel(props.commandInput)}
        </FormLabel>
        <Component {...props} />
        <FormHelperText error={props.meta.touched && !!props.meta.error}>
            {
                props.meta.touched && props.meta.error
                    ? props.meta.error
                    : props.commandInput.doc
            }
        </FormHelperText>
    </FormGroup>;
};