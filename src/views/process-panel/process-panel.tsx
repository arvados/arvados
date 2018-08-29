// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from '~/views/process-panel/information-card';
import { SubprocessesCard } from '~/views/process-panel/subprocesses-card';

export class ProcessPanel extends React.Component {
    render() {
        return <Grid container spacing={16}>
            <Grid item xs={7}>
                <ProcessInformationCard />
            </Grid>
            <Grid item xs={5}>
                <SubprocessesCard />
            </Grid>
        </Grid>;
    }
}
