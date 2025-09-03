// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';
import moment, { Moment } from 'moment';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';

type MomentProps = {
        num: number;
        unit: 'year' | 'month' | 'week' | 'day' | 'hour' | 'minute' | 'second';
    }

type DatePickerProps = {
    label: string;
    minDate?: MomentProps;
}

export function DatePicker({label, minDate}: DatePickerProps) {
    const [value, setValue] = React.useState<Moment | null>(minDate ? moment().add(minDate.num, minDate.unit) : moment());

    return (
        <LocalizationProvider dateAdapter={AdapterMoment}>
            <DesktopDatePicker
                label={label}
                value={value}
                minDate={minDate ? moment().add(minDate.num, minDate.unit) : moment()}
                onChange={(newValue) => {
                    setValue(newValue);
                }}
            />
        </LocalizationProvider>
    );
}
