// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProcessInformationCard } from '~/views/process-panel/information-card';
import { Grid } from '@material-ui/core';

export class ProcessPanel extends React.Component {
    render() {
        return <div>
            <Grid container>
                <Grid item xs={7}>
                    <ProcessInformationCard />
                </Grid>
            </Grid>
        </div>;
    }
}