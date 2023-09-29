// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CommandInputParameter } from 'models/workflow';
import { TextField } from '@material-ui/core';
import { required } from 'components/workflow-inputs-form/validators';

export interface WorkflowInputProps {
    input: CommandInputParameter;
}
export const WorkflowInput = ({ input }: WorkflowInputProps) =>
    <TextField
        label={`${input.label || input.id}${required(input)() ? '*' : ''}`}
        name={input.id}
        helperText={input.doc}
        fullWidth />;