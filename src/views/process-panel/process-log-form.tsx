// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    withStyles,
    WithStyles,
    StyleRulesCallback,
    FormControl,
    Select,
    MenuItem,
    Input
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'formControl';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    formControl: {
        minWidth: theme.spacing.unit * 15,
    }
});

export interface FilterOption {
    label: string;
    value: string;
}

export interface ProcessLogFormDataProps {
    selectedFilter: FilterOption;
    filters: FilterOption[];
}

export interface ProcessLogFormActionProps {
    onChange: (filter: FilterOption) => void;
}

type ProcessLogFormProps = ProcessLogFormDataProps & ProcessLogFormActionProps & WithStyles<CssRules>;

export const ProcessLogForm = withStyles(styles)(
    ({ classes, selectedFilter, onChange, filters }: ProcessLogFormProps) =>
        <form autoComplete="off" data-cy="process-logs-filter">
            <FormControl className={classes.formControl}>
                <Select
                    value={selectedFilter.value}
                    onChange={({ target }) => onChange({ label: target.innerText, value: target.value })}
                    input={<Input name="eventType" id="log-label-placeholder" />}
                    name="eventType">
                    {
                        filters.map(option =>
                            <MenuItem key={option.value} value={option.value}>{option.label}</MenuItem>
                        )
                    }
                </Select>
            </FormControl>
        </form>
);