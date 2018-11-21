// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { SSH_KEY_PUBLIC_VALIDATION, SSH_KEY_NAME_VALIDATION } from "~/validators/validators";

export const SshKeyPublicField = () =>
    <Field
        name='publicKey'
        component={TextField}
        validate={SSH_KEY_PUBLIC_VALIDATION}
        autoFocus={true}
        label="Public Key" />;

export const SshKeyNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={SSH_KEY_NAME_VALIDATION}
        label="Name" />;


