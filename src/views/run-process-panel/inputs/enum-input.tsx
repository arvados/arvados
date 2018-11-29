// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Field } from 'redux-form';
import { Select, MenuItem } from '@material-ui/core';
import { EnumCommandInputParameter, CommandInputEnumSchema } from '~/models/workflow';
import { GenericInputProps, GenericInput } from './generic-input';

export interface EnumInputProps {
    input: EnumCommandInputParameter;
}
export const EnumInput = ({ input }: EnumInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={EnumInputComponent}
    />;

const EnumInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) => {
    const type = props.commandInput.type as CommandInputEnumSchema;
    return <Select
        value={props.input.value}
        onChange={props.input.onChange}
        disabled={props.commandInput.disabled} >
        {type.symbols.map(symbol =>
            <MenuItem key={symbol} value={extractValue(symbol)}>
                {extractValue(symbol)}
            </MenuItem>)}
    </Select>;
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
