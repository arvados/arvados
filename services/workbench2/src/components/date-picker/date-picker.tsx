// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import { WrappedFieldProps } from 'redux-form';
import { FormControl } from '@mui/material';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';
import moment from 'moment';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';

type DatePickerProps = {
    label: string;
    startValue?: string;
}

export function DatePicker({label, startValue, input}: DatePickerProps & WrappedFieldProps) {

    // initial value doesn't automatically get passed to redux-form,
    // so we need to set it manually on mount
    useEffect(() => {
            input.onChange(getInitialValue(startValue, input.value));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
        <FormControl variant="standard" fullWidth>
            <LocalizationProvider dateAdapter={AdapterMoment}>
                <DesktopDatePicker
                    disablePast
                    label={label}
                    value={getInitialValue(startValue, input.value)}
                    onChange={input.onChange}
                />
            </LocalizationProvider>
        </FormControl>
    );
}


const getInitialValue = (startValue: string | undefined, inputValue: string | undefined) => {
    if (inputValue) { // Set by the user
        return moment(inputValue);
    }
    if (startValue) { // Passed in as a prop
        return moment(startValue);
    }
    // If no value is set yet and no startValue is passed in, use today
    return moment();
};