// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Field } from "redux-form";
import moment from "moment";
import { TextField, RichEditorTextField, TextFieldWithStartValue } from "components/text-field/text-field";
import { REQUIRED_VALIDATION, LENGTH255_VALIDATION, REQUIRED_LENGTH255_VALIDATION, REQUIRED_VALIDNAME_LENGTH255_VALIDATION, DATE_VALIDATION } from "validators/validators";
import { DatePicker } from "components/date-picker/date-picker";
import { StringArrayInput } from "components/string-array-input/string-array-input";

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

export const ExternalCredentialClassCreateField = () =>
    <Field
        name='credentialClass'
        component={TextFieldWithStartValue as any}
        startValue={'aws_access_key'}
        validate={REQUIRED_LENGTH255_VALIDATION}
        label="Credential Class *" />;

export const ExternalCredentialClassUpdateField = () =>
    <Field
        name='credentialClass'
        component={TextField as any}
        validate={REQUIRED_LENGTH255_VALIDATION}
        label="Credential Class *" />;

export const ExternalCredentialExternalIdField = () =>
    <Field
        name='externalId'
        component={TextField as any}
        validate={REQUIRED_LENGTH255_VALIDATION}
        label="External ID *" />;

export const ExternalCredentialExpiresAtField = () =>
    <Field
        name='expiresAt'
        component={DatePicker as any}
        startValue={moment().add(1, 'year')}
        validate={DATE_VALIDATION}
        label="Expires at" />;

export const ExternalCredentialSecretCreateField = () =>
    <Field
        name='secret'
        component={TextField as any}
        type='password'
        autoComplete="new-password"
        validate={REQUIRED_VALIDATION}
        label="Secret *" />;

export const ExternalCredentialSecretUpdateField = () =>
    <Field
        name='secret'
        component={TextField as any}
        type='password'
        autoComplete="new-password"
        helperText="Leave blank to keep the same secret"
        label="Secret" />;

export const ExternalCredentialScopesField = () =>
        <Field
            name="scopes"
            component={StringArrayInput as any}
            label="Applicable scopes"
        />
