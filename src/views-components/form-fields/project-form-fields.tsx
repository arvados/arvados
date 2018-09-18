// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { PROJECT_NAME_VALIDATION, PROJECT_DESCRIPTION_VALIDATION } from "~/validators/validators";

export const ProjectNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={PROJECT_NAME_VALIDATION}
        label="Project Name"
        autoFocus={true} />;

export const ProjectDescriptionField = () =>
    <Field
        name='description'
        component={TextField}
        validate={PROJECT_DESCRIPTION_VALIDATION}
        label="Description - optional" />;
