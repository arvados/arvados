// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from 'redux-form';
import { TextField, DataTextField } from "~/components/text-field/text-field";
import { CheckboxField } from '~/components/checkbox-field/checkbox-field';
import { NativeSelectField } from '~/components/select-field/select-field';
import { ResourceKind } from '~/models/resource';
import { ClusterObjectType } from '~/models/search-bar';

export const SearchBarTypeField = () =>
    <Field
        name='type'
        component={NativeSelectField}
        items={[
            { key: '', value: 'Any'},
            { key: ResourceKind.COLLECTION, value: 'Collection'},
            { key: ResourceKind.PROJECT, value: 'Project' },
            { key: ResourceKind.PROCESS, value: 'Process' }
        ]}/>;

export const SearchBarClusterField = () =>
    <Field
        name='cluster'
        component={NativeSelectField}
        items={[
            { key: '', value: 'Any' },
            { key: ClusterObjectType.INDIANAPOLIS, value: 'Indianapolis' },
            { key: ClusterObjectType.KAISERAUGST, value: 'Kaiseraugst' },
            { key: ClusterObjectType.PENZBERG, value: 'Penzberg' }
        ]} />;

export const SearchBarProjectField = () => 
    <div>Box</div>;

export const SearchBarTrashField = () => 
    <Field
        name='inTrash'
        component={CheckboxField}
        label="In trash" />;

export const SearchBarDataFromField = () => 
    <Field
        name='dateFrom'
        component={DataTextField} />;

export const SearchBarDataToField = () =>
    <Field
        name='dateTo'
        component={DataTextField} />;

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
    <Field
        name='saveQuery'
        component={CheckboxField}
        label="Save search query" />;

export const SearchBarQuerySearchField = () => 
    <Field
        name='searchQuery'
        component={TextField}
        label="Search query name" />;