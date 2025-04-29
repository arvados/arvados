// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { FormControl, FormControlLabel, Radio, RadioGroup } from '@mui/material';
import { WrappedFieldProps } from 'redux-form';
import { ArvadosTheme, CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';

type CssRules = 'radioGroupRow';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    radioGroupRow: {
        flexDirection: 'row',
    },
});

interface RadioFieldDataProps {
    items: {key: string, value: any}[],
    flexRowDirection: boolean,
}

export const RadioField = withStyles(styles)((props: WrappedFieldProps & RadioFieldDataProps & WithStyles<CssRules>) =>
    <FormControl>
        <RadioGroup
            className={props.flexRowDirection ? props.classes.radioGroupRow : undefined}
            name={props.input.name}
            value={props.input.value}
            onChange={props.input.onChange}
        >
            {props.items.map(item => (
                <FormControlLabel key={item.key} value={item.key} control={<Radio />} label={item.value} />
            ))}
        </RadioGroup>
    </FormControl>);
