// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { FormFieldProps, FormField } from 'components/form-field/form-field';
import { Switch } from '@mui/material';
import { SwitchProps } from '@mui/material/Switch';

export const SwitchField = ({ switchProps, ...props }: FormFieldProps & { switchProps: SwitchProps }) =>
    <FormField {...props}>
        {input => <Switch {...switchProps} checked={input.value} onChange={input.onChange} />}
    </FormField>;

