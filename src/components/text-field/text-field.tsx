// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { ArvadosTheme } from '../../common/custom-theme';
import { TextField as MaterialTextField, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type CssRules = 'textField';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    textField: {
        marginBottom: theme.spacing.unit * 3
    },
});

export const TextField = withStyles(styles)((props: WrappedFieldProps & WithStyles<CssRules> & { label?: string }) =>
    <MaterialTextField
        helperText={props.meta.touched && props.meta.error}
        className={props.classes.textField}
        label={props.label}
        disabled={props.meta.submitting}
        error={props.meta.touched && !!props.meta.error}
        autoComplete='off'
        fullWidth={true}
        {...props.input}
    />);