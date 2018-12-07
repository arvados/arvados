// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION, COLLECTION_PROJECT_VALIDATION } from "~/validators/validators";
import { ProjectTreePickerField } from "~/views-components/project-tree-picker/project-tree-picker";
import { PickerIdProp } from '~/store/tree-picker/picker-id';

export const CollectionNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={COLLECTION_NAME_VALIDATION}
        label="Collection Name"
        autoFocus={true} />;

export const CollectionDescriptionField = () =>
    <Field
        name='description'
        component={TextField}
        validate={COLLECTION_DESCRIPTION_VALIDATION}
        label="Description - optional" />;

export const CollectionProjectPickerField = (props: PickerIdProp) =>
    <Field
        name="projectUuid"
        pickerId={props.pickerId}
        component={ProjectTreePickerField}
        validate={COLLECTION_PROJECT_VALIDATION} />;
