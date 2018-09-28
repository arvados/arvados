// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IntCommandInputParameter, getInputLabel } from '~/models/workflow';
import { Field } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { isInteger } from '~/validators/is-integer';

export interface IntInputProps {
    input: IntCommandInputParameter;
}
export const IntInput = ({ input }: IntInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={TextField}
        parse={value => parseInt(value, 10)}
        format={value => isNaN(value) ? '' : JSON.stringify(value)}
        validate={[isInteger]} />;

