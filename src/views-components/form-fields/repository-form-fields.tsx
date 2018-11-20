// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { REPOSITORY_NAME_VALIDATION } from "~/validators/validators";

export const RepositoryNameField = () =>
    <span>
        pawelkowalczyk/
        <Field
            name='name'
            component={TextField}
            validate={REPOSITORY_NAME_VALIDATION}
            label="Name"
            autoFocus={true} />.git<br/>
            It may take a minute or two before you can clone your new repository.
            </span>;