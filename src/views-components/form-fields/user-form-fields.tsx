// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { USER_EMAIL_VALIDATION, USER_LENGTH_VALIDATION } from "~/validators/validators";
import { NativeSelectField } from "~/components/select-field/select-field";

export const UserFirstNameField = () =>
    <Field
        name='firstName'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        autoFocus={true}
        label="First name" />;

export const UserLastNameField = () =>
    <Field
        name='lastName'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        autoFocus={true}
        label="Last name" />;

export const UserEmailField = () =>
    <Field
        name='email'
        component={TextField}
        validate={USER_EMAIL_VALIDATION}
        autoFocus={true}
        label="Email" />;

export const UserIdentityUrlField = () =>
    <Field
        name='identityUrl'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        label="Identity URL Prefix" />;

export const UserVirtualMachineField = () =>
    <Field
        name='virtualMachine'
        component={NativeSelectField}
        validate={USER_LENGTH_VALIDATION}
        items={['shell']} />;

export const UserGroupsVirtualMachineField = () =>
    <Field
        name='virtualMachine'
        component={TextField}
        validate={USER_LENGTH_VALIDATION}
        label="Groups for virtual machine (comma separated list)" />;