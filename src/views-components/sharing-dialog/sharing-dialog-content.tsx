// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';

import { SharingInvitationForm } from './sharing-invitation-form';
import { SharingManagementForm } from './sharing-management-form';
import { Grid } from '@material-ui/core';

export const SharingDialogContent = (props: { advancedViewOpen: boolean }) =>
    <Grid container direction='column' spacing={24}>
        {props.advancedViewOpen &&
            <>
                <Grid item>
                    <SharingManagementForm />
                </Grid>
            </>
        }
        <Grid item>
            <SharingInvitationForm />
        </Grid>
    </Grid>;
