// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { getInputLabel, FloatCommandInputParameter } from '~/models/workflow';
import { Field } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { isNumber } from '~/validators/is-number';
import { toNumber } from 'lodash';
export interface FloatInputProps {
    input: FloatCommandInputParameter;
}
export const FloatInput = ({ input }: FloatInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={TextField}
        parse={value => toNumber(value)}
        format={value => isNaN(value) ? '' : JSON.stringify(value)}
        validate={[isNumber]} />;

