// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Typography } from "@material-ui/core";
import { TreeItem } from "~/components/tree/tree";
import { WrappedFieldProps } from 'redux-form';
import { ProjectsTreePicker } from '~/views-components/projects-tree-picker/projects-tree-picker';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { PickerIdProp } from '~/store/tree-picker/picker-id';

export const ProjectTreePickerField = (props: WrappedFieldProps & PickerIdProp) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectsTreePicker
            pickerId={props.pickerId}
            toggleItemActive={handleChange(props)}
            options={{ showOnlyOwned: false, showOnlyWritable: true }} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) =>
    (_: any, { id }: TreeItem<ProjectsTreePickerItem>) =>
        props.input.onChange(id);

export const CollectionTreePickerField = (props: WrappedFieldProps & PickerIdProp) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectsTreePicker
            pickerId={props.pickerId}
            toggleItemActive={handleChange(props)}
            options={{ showOnlyOwned: false, showOnlyWritable: true }}
            includeCollections />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;