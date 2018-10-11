// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { Checkbox, FormControlLabel } from '@material-ui/core';

export const SearchBarTypeField = () =>
    <Field
        name='type'
        component={TextField}
        label="Type"/>;

export const SearchBarClusterField = () =>
    <Field
        name='cluster'
        component={TextField}
        label="Cluster name" />;

export const SearchBarProjectField = () => 
    <Field
        name='project'
        component={TextField}
        label="Project name" />;

export const SearchBarTrashField = () => 
    <FormControlLabel
        control={
            <Checkbox
                checked={false}
                value="true"
                color="primary"
            />
        }
        label="In trash" />;

export const SearchBarDataFromField = () => 
    <Field
        name='dataFrom'
        component={TextField}
        label="From" />;

export const SearchBarDataToField = () =>
    <Field
        name='dataTo'
        component={TextField}
        label="To" />;

export const SearchBarKeyField = () => 
    <Field
        name='key'
        component={TextField}
        label="Key" />;

export const SearchBarValueField = () => 
    <Field
        name='value'
        component={TextField}
        label="Value" />;

export const SearchBarSaveSearchField = () => 
    <FormControlLabel
        control={
            <Checkbox
                checked={true}
                value="true"
                color="primary"
            />
        }
        label="Save search query" />;

export const SearchBarQuerySearchField = () => 
    <Field
        name='searchQuery'
        component={TextField}
        label="Search query name" />;