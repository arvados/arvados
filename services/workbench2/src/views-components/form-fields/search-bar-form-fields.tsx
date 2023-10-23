// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field, FieldArray } from 'redux-form';
import { TextField, DateTextField } from "components/text-field/text-field";
import { CheckboxField } from 'components/checkbox-field/checkbox-field';
import { NativeSelectField } from 'components/select-field/select-field';
import { ResourceKind } from 'models/resource';
import { SearchBarAdvancedPropertiesView } from 'views-components/search-bar/search-bar-advanced-properties-view';
import { PropertyKeyField, } from 'views-components/resource-properties-form/property-key-field';
import { PropertyValueField } from 'views-components/resource-properties-form/property-value-field';
import { connect } from "react-redux";
import { RootState } from "store/store";
import { ProjectInput, ProjectCommandInputParameter } from 'views/run-process-panel/inputs/project-input';

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
    <ProjectInput required={false} input={{
        id: "projectObject",
        label: "Limit search to Project"
    } as ProjectCommandInputParameter}
        options={{ showOnlyOwned: false, showOnlyWritable: false }} />

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
