// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field, Validator } from "redux-form";
import { TextField, RichEditorTextField } from "~/components/text-field/text-field";
import { PROJECT_NAME_VALIDATION, PROJECT_NAME_VALIDATION_ALLOW_SLASH } from "~/validators/validators";
import { connect } from "react-redux";
import { RootState } from "~/store/store";

interface ProjectNameFieldProps {
    validate: Validator[];
    label?: string;
}

// Validation behavior depends on the value of ForwardSlashNameSubstitution.
//
// Redux form doesn't let you pass anonymous functions to 'validate'
// -- it fails with a very confusing recursive-update-exceeded error.
// So we can't construct the validation function on the fly.
//
// As a workaround, use ForwardSlashNameSubstitution to choose between one of two const-defined validators.

export const ProjectNameField = connect(
    (state: RootState) => {
        return {
            validate: (state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution === "" ?
                PROJECT_NAME_VALIDATION : PROJECT_NAME_VALIDATION_ALLOW_SLASH)
        };
    })((props: ProjectNameFieldProps) =>
        <span data-cy='name-field'><Field
            name='name'
            component={TextField}
            validate={props.validate}
            label={props.label || "Project Name"}
            autoFocus={true} /></span>
    );

export const ProjectDescriptionField = () =>
    <Field
        name='description'
        component={RichEditorTextField}
        label="Description - optional" />;
