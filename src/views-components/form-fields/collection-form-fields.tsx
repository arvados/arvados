// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field, WrappedFieldProps } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION, COLLECTION_PROJECT_VALIDATION } from "~/validators/validators";
import { ProjectTreePicker } from "~/views-components/project-tree-picker/project-tree-picker";

export const CollectionNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={COLLECTION_NAME_VALIDATION}
        label="Collection Name" />;

export const CollectionDescriptionField = () =>
    <Field
        name='description'
        component={TextField}
        validate={COLLECTION_DESCRIPTION_VALIDATION}
        label="Description - optional" />;

export const CollectionProjectPickerField = () =>
    <Field
        name="projectUuid"
        component={ProjectPicker}
        validate={COLLECTION_PROJECT_VALIDATION} />;

const ProjectPicker = (props: WrappedFieldProps) =>
    <div style={{ width: '400px', height: '144px', display: 'flex', flexDirection: 'column' }}>
        <ProjectTreePicker onChange={projectUuid => props.input.onChange(projectUuid)} />
    </div>;
