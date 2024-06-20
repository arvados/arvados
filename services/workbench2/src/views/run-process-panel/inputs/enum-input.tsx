// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field } from 'redux-form';
import { memoize } from 'lodash/fp';
import { fieldRequire } from 'validators/require';
import { Select, MenuItem } from '@mui/material';
import { EnumCommandInputParameter, CommandInputEnumSchema, isRequiredInput, getEnumType } from 'models/workflow';
import { GenericInputProps, GenericInput } from './generic-input';

export interface EnumInputProps {
    input: EnumCommandInputParameter;
}

const getValidation = memoize(
    (input: EnumCommandInputParameter) => ([
        isRequiredInput(input)
            ? fieldRequire
            : () => undefined,
    ]));

const emptyToNull = value => {
    if (value === '') {
        return null;
    } else {
        return value;
    }
};

export const EnumInput = ({ input }: EnumInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={EnumInputComponent}
        validate={getValidation(input)}
        normalize={emptyToNull}
    />;

const EnumInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) => {
    const type = getEnumType(props.commandInput) as CommandInputEnumSchema;
    return (
        <Select
            variant="standard"
            value={props.input.value}
            onChange={props.input.onChange}
            disabled={props.commandInput.disabled}>
            {(isRequiredInput(props.commandInput) ? [] : [<MenuItem key={'_empty'} value={''} />]).concat(type.symbols.map(symbol =>
                <MenuItem key={symbol} value={extractValue(symbol)}>
                    {extractValue(symbol)}
                </MenuItem>))}
        </Select>
    );
};

/**
 * Values in workflow definition have an absolute form, for example:
 *
 * ```#input_collector.cwl/enum_type/Pathway table```
 *
 * We want a value that is in form accepted by backend.
 * According to the example above, the correct value is:
 *
 * ```Pathway table```
 */
const extractValue = (symbol: string) => symbol.split('/').pop();
