// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, WithStyles, withStyles, FormControl, InputLabel, Select, MenuItem } from '@material-ui/core';

type CssRules = 'formControl' | 'selectWrapper' | 'select' | 'option';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    formControl: {
        width: '100%'
    },
    selectWrapper: {
        backgroundColor: 'white',
        '&:before': {
            borderBottomColor: 'rgba(0, 0, 0, 0.42)'
        },
        '&:focus': {
            outline: 'none'
        }
    },
    select: {
        fontSize: '0.875rem',
        '&:focus': {
            backgroundColor: 'rgba(0, 0, 0, 0.0)'
        }
    },
    option: {
        fontSize: '0.875rem',
        backgroundColor: 'white',
        height: '30px'
    }
});

export const NativeSelectField = withStyles(styles)
    ((props: WrappedFieldProps & WithStyles<CssRules> & { items: any[] }) =>
        <FormControl className={props.classes.formControl}>
            <Select className={props.classes.selectWrapper}
                native
                value={props.input.value}
                onChange={props.input.onChange}
                disabled={props.meta.submitting}
                name={props.input.name}
                inputProps={{
                    id: `id-${props.input.name}`,
                    className: props.classes.select
                }}>
                {props.items.map(item => (
                    <option key={item.key} value={item.key} className={props.classes.option}>
                        {item.value}
                    </option>
                ))}
            </Select>
        </FormControl>
    );