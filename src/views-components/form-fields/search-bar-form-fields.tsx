// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field, WrappedFieldProps, FieldArray } from 'redux-form';
import { TextField, DateTextField } from "components/text-field/text-field";
import { CheckboxField } from 'components/checkbox-field/checkbox-field';
import { NativeSelectField } from 'components/select-field/select-field';
import { ResourceKind } from 'models/resource';
import { HomeTreePicker } from 'views-components/projects-tree-picker/home-tree-picker';
import { SEARCH_BAR_ADVANCED_FORM_PICKER_ID } from 'store/search-bar/search-bar-actions';
import { SearchBarAdvancedPropertiesView } from 'views-components/search-bar/search-bar-advanced-properties-view';
import { TreeItem } from "components/tree/tree";
import { ProjectsTreePickerItem } from "store/tree-picker/tree-picker-middleware";
import { PropertyKeyField, } from 'views-components/resource-properties-form/property-key-field';
import { PropertyValueField } from 'views-components/resource-properties-form/property-value-field';
import { connect } from "react-redux";
import { RootState } from "store/store";

export const SearchBarTypeField = () =>
    <Field
        name='type'
        component={NativeSelectField as any}
        items={[
            { key: '', value: 'Any' },
            { key: ResourceKind.COLLECTION, value: 'Collection' },
            { key: ResourceKind.PROJECT, value: 'Project' },
            { key: ResourceKind.PROCESS, value: 'Process' }
        ]} />;


interface SearchBarClusterFieldProps {
    clusters: { key: string, value: string }[];
}

export const SearchBarClusterField = connect(
    (state: RootState) => ({
        clusters: [{ key: '', value: 'Any' }].concat(
            state.auth.sessions
                .filter(s => s.loggedIn)
                .map(s => ({
                    key: s.clusterId,
                    value: s.clusterId
                })))
    }))((props: SearchBarClusterFieldProps) => <Field
        name='cluster'
        component={NativeSelectField as any}
        items={props.clusters} />
    );

export const SearchBarProjectField = () =>
    <Field
        name='projectUuid'
        component={ProjectsPicker} />;

const ProjectsPicker = (props: WrappedFieldProps) =>
    <div style={{ height: '100px', display: 'flex', flexDirection: 'column', overflow: 'overlay' }}>
        <HomeTreePicker
            pickerId={SEARCH_BAR_ADVANCED_FORM_PICKER_ID}
            toggleItemActive={
                (_: any, { id }: TreeItem<ProjectsTreePickerItem>) => {
                    props.input.onChange(id);
                }
            } />
    </div>;

export const SearchBarTrashField = () =>
    <Field
        name='inTrash'
        component={CheckboxField}
        label="In trash" />;

export const SearchBarPastVersionsField = () =>
    <Field
        name='pastVersions'
        component={CheckboxField}
        label="Past versions" />;

export const SearchBarDateFromField = () =>
    <Field
        name='dateFrom'
        component={DateTextField as any} />;

export const SearchBarDateToField = () =>
    <Field
        name='dateTo'
        component={DateTextField as any} />;

export const SearchBarPropertiesField = () =>
    <FieldArray
        name="properties"
        component={SearchBarAdvancedPropertiesView as any} />;

export const SearchBarKeyField = () =>
    <PropertyKeyField skipValidation={true} />;

export const SearchBarValueField = () =>
    <PropertyValueField skipValidation={true} />;

export const SearchBarSaveSearchField = () =>
    <Field
        name='saveQuery'
        component={CheckboxField}
        label="Save query" />;

export const SearchBarQuerySearchField = () =>
    <Field
        name='queryName'
        component={TextField as any}
        label="Query name" />;
