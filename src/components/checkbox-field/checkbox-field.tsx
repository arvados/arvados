// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps } from 'redux-form';
import { FormControlLabel, Checkbox } from '@material-ui/core';

export const CheckboxField = (props: WrappedFieldProps & { label?: string }) =>
    <FormControlLabel
        control={
            <Checkbox
                checked={props.input.value}
                onChange={props.input.onChange}
                disabled={props.meta.submitting}
                color="primary" />
        }
        label={props.label} 
    />;