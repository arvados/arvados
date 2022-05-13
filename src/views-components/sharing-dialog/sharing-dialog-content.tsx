// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Grid } from '@material-ui/core';

import { SharingManagementForm } from './sharing-management-form';

export const SharingDialogContent = () =>
    <Grid container direction='column' spacing={24}>
        <Grid item>
            <SharingManagementForm />
        </Grid>
    </Grid>;
