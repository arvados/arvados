// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { ArvadosTheme } from '~/common/custom-theme';
import { FormControlLabel, Checkbox, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type CssRules = 'checkboxField';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    checkboxField: {
        
    }
});

export const CheckboxField = withStyles(styles)((props: WrappedFieldProps & WithStyles<CssRules> & { label?: string }) =>
    <FormControlLabel
        control={
            <Checkbox
                checked={props.input.value}
                onChange={props.input.onChange}
                color="primary" />
        }
        label={props.label} 
    />);