// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field, WrappedFieldProps } from 'redux-form';
import { TextField, DateTextField } from "~/components/text-field/text-field";
import { CheckboxField } from '~/components/checkbox-field/checkbox-field';
import { NativeSelectField } from '~/components/select-field/select-field';
import { ResourceKind } from '~/models/resource';
import { ClusterObjectType } from '~/models/search-bar';
import { HomeTreePicker } from '~/views-components/projects-tree-picker/home-tree-picker';
import { SEARCH_BAR_ADVANCE_FORM_PICKER_ID } from '~/store/search-bar/search-bar-actions';

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
    <Field
        name='projectUuid'
        component={ProjectsPicker} />;

const ProjectsPicker = (props: WrappedFieldProps) =>
    <div style={{ height: '100px', display: 'flex', flexDirection: 'column', overflow: 'overlay' }}>
        <HomeTreePicker pickerId={SEARCH_BAR_ADVANCE_FORM_PICKER_ID} />
    </div>;

export const SearchBarTrashField = () => 
    <Field
        name='inTrash'
        component={CheckboxField}
        label="In trash" />;

export const SearchBarDataFromField = () => 
    <Field
        name='dateFrom'
        component={DateTextField} />;

export const SearchBarDataToField = () =>
    <Field
        name='dateTo'
        component={DateTextField} />;

export const SearchBarKeyField = () => 
    <Field
        name='propertyKey'
        component={TextField}
        label="Key" />;

export const SearchBarValueField = () => 
    <Field
        name='propertyValue'
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