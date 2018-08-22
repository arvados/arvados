// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ProjectTreePicker as ProjectPicker } from "~/views-components/project-tree-picker/project-tree-picker";
import { WrappedFieldProps } from "redux-form";
import { Typography } from '@material-ui/core';

export const ProjectTreePicker = (props: WrappedFieldProps) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectPicker onChange={handleChange(props)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) => (value: string) =>
    props.input.value === value
        ? props.input.onChange('')
        : props.input.onChange(value);