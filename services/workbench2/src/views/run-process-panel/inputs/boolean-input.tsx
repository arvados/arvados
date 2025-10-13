// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { memoize } from 'lodash/fp';
import { BooleanCommandInputParameter } from 'models/workflow';
import { Field } from 'redux-form';
import { Switch, Theme } from '@mui/material';
import { GenericInputProps, GenericInput } from './generic-input';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';

export interface BooleanInputProps {
    input: BooleanCommandInputParameter;
}
export const BooleanInput = ({ input }: BooleanInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={BooleanInputComponent}
        normalize={normalize}
    />;

const normalize = (_: any, prevValue: boolean) => !prevValue;

const BooleanInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

type CssRules = "switch";

const styles: CustomStyleRulesCallback<CssRules> = (theme: Theme) => ({
    switch: {
        marginTop: "15px",
        marginBottom: "-9px", // To line up hint text with GenericInput
    },
});

const Input = withStyles(styles)(({ input, commandInput, classes }: GenericInputProps & WithStyles<CssRules>) =>
    <Switch
        color='primary'
        className={classes.switch}
        checked={input.value}
        onChange={handleChange(input.onChange, input.value)}
        disabled={commandInput.disabled}
    />
);

const handleChange = memoize(
    (onChange: (value: string) => void, value: string) => () => onChange(value)
);
