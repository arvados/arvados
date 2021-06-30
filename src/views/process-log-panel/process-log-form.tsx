// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { withStyles, WithStyles, StyleRulesCallback, FormControl, InputLabel, Select, MenuItem, Input } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { FilterOption } from './process-log-panel';

type CssRules = 'formControl';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    formControl: {
        minWidth: 200
    }
});

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
        <form autoComplete="off">
            <FormControl className={classes.formControl}>
                <InputLabel shrink htmlFor="log-label-placeholder">
                    Event Type
                </InputLabel>
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