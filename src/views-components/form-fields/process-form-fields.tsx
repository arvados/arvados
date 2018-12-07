// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { PROCESS_NAME_VALIDATION } from "~/validators/validators";

export const ProcessNameField = () =>
    <Field
        name='name'
        component={TextField}
        validate={PROCESS_NAME_VALIDATION}
        label="Process Name" />;
