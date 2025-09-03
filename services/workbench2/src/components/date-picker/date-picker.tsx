// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';
import moment, { Moment } from 'moment';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';

export function DatePicker(label: string) {
    const [value, setValue] = React.useState<Moment | null>(moment().add(1, 'year'));

    return (
        <LocalizationProvider dateAdapter={AdapterMoment}>
            <DesktopDatePicker
                label={label}
                value={value}
                minDate={moment().add(1, 'year')}
                onChange={(newValue) => {
                    setValue(newValue);
                }}
            />
        </LocalizationProvider>
    );
}
