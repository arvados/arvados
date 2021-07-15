// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps } from 'redux-form';
import {
    FormControlLabel,
    Checkbox,
    FormControl,
    FormGroup,
    FormLabel,
    FormHelperText
} from '@material-ui/core';

export const CheckboxField = (props: WrappedFieldProps & { label?: string }) =>
    <FormControlLabel
        control={
            <Checkbox
                checked={props.input.value}
                onChange={props.input.onChange}
                disabled={props.meta.submitting}
                color="primary" />
        }
        label={props.label}
    />;

type MultiCheckboxFieldProps = {
    items: string[];
    label?: string;
    minSelection?: number;
    maxSelection?: number;
    helperText?: string;
    rowLayout?: boolean;
}

export const MultiCheckboxField = (props: WrappedFieldProps & MultiCheckboxFieldProps) => {
    const isValid = (items: string[]) => (items.length >= (props.minSelection || 0)) &&
        (items.length <= (props.maxSelection || items.length));
    return <FormControl error={!isValid(props.input.value)}>
        <FormLabel component='label'>{props.label}</FormLabel>
        <FormGroup row={props.rowLayout}>
        { props.items.map((item, idx) =>
            <FormControlLabel
                control={
                    <Checkbox
                        data-cy={`checkbox-${item}`}
                        key={idx}
                        name={`${props.input.name}[${idx}]`}
                        value={item}
                        checked={props.input.value.indexOf(item) !== -1}
                        onChange={e => {
                            const newValue = [...props.input.value];
                            if (e.target.checked) {
                                newValue.push(item);
                            } else {
                                newValue.splice(newValue.indexOf(item), 1);
                            }
                            if (!isValid(newValue)) { return; }
                            return props.input.onChange(newValue);
                        }}
                        disabled={props.meta.submitting}
                        color="primary" />
                }
                label={item} />) }
        </FormGroup>
        <FormHelperText>{props.helperText}</FormHelperText>
    </FormControl> };