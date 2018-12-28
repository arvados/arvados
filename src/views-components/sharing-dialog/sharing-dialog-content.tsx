// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Typography } from '@material-ui/core';

import { SharingInvitationForm } from './sharing-invitation-form';
import { SharingManagementForm } from './sharing-management-form';
import { SharingPublicAccessForm } from './sharing-public-access-form';

export const SharingDialogContent = (props: { advancedViewOpen: boolean }) =>
    <Grid container direction='column' spacing={24}>
        {props.advancedViewOpen &&
            <>
                <Grid item>
                    <Typography variant='subtitle1'>
                        Who can access
                    </Typography>
                    <SharingPublicAccessForm />
                    <SharingManagementForm />
                </Grid>
            </>
        }
        <Grid item>
            <SharingInvitationForm />
        </Grid>
    </Grid>;
