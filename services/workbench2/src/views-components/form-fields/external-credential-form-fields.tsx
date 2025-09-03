// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field } from "redux-form";
import { TextField, RichEditorTextField } from "components/text-field/text-field";
import { REQUIRED_LENGTH255_VALIDATION, REQUIRED_VALIDNAME_LENGTH255_VALIDATION, LENGTH255_VALIDATION } from "validators/validators";
import { DatePicker } from "components/date-picker/date-picker";

export const ExternalCredentialNameField = () =>
    <Field
        name='name'
        component={TextField as any}
        validate={REQUIRED_VALIDNAME_LENGTH255_VALIDATION}
        label={"Credential Name *"}
        autoFocus={true} />;

export const ExternalCredentialDescriptionField = () =>
    <Field
        name='description'
        component={RichEditorTextField as any}
        validate={LENGTH255_VALIDATION}
        label="Description" />;

export const ExternalCredentialClassField = () =>
    <Field
        name='credentialClass'
        component={TextField as any}
        validate={LENGTH255_VALIDATION}
        label="Credential Class" />;

export const ExternalCredentialExternalIdField = () =>
    <Field
        name='externalId'
        component={TextField as any}
        validate={REQUIRED_LENGTH255_VALIDATION}
        label="External ID *" />;

export const ExternalCredentialExpiresAtField = () =>
    <Field
        name='expiresAt'
        component={()=>DatePicker({label: 'Expires at', minDate: {num: 1, unit: 'week'}})} />;

export const ExternalCredentialScopesField = () =>
    <Field
        name='scopes'
        component={TextField as any}
        validate={LENGTH255_VALIDATION}
        helperText="Comma separated list of scopes"
        label="Scopes" />;


