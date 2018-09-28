// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { getInputLabel, isRequiredInput, StringCommandInputParameter } from '~/models/workflow';
import { Field } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { require } from '~/validators/require';

export interface StringInputProps {
    input: StringCommandInputParameter;
}
export const StringInput = ({ input }: StringInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={TextField}
        validate={[
            isRequiredInput(input)
                ? require
                : () => undefined,
        ]} />;

