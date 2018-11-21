// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { REPOSITORY_NAME_VALIDATION } from "~/validators/validators";
import { Grid } from "@material-ui/core";

export const RepositoryNameField = (props: any) =>
    <Grid container style={{ marginTop: '0', paddingTop: '24px' }}>
        <Grid item xs={3}>
            {props.data.user.username}/
        </Grid>
        <Grid item xs={7} style={{ bottom: '24px', position: 'relative' }}>
            <Field
                name='name'
                component={TextField}
                validate={REPOSITORY_NAME_VALIDATION}
                label="Name"
                autoFocus={true} />
        </Grid>
        <Grid item xs={2}>
            .git
        </Grid>
        <Grid item xs={12}>
            It may take a minute or two before you can clone your new repository.
        </Grid>
    </Grid>;