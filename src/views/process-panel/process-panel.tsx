// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { } from '@material-ui/core';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from '~/views-components/process-information-card/process-information-card';
import { ProcessSubprocesses } from '~/views-components/process-subprocesses/process-subprocesses';
import { SubprocessesStatus } from '~/views/process-panel/process-subprocesses';

export type CssRules = 'headerActive' | 'headerCompleted' | 'headerQueued' | 'headerFailed' | 'headerCanceled';

export class ProcessPanel extends React.Component {
    render() {
        return <div>
            <Grid container>
                <Grid item xs={7}>
                    <ProcessInformationCard />
                </Grid>
            </Grid>
            <ProcessSubprocesses />
        </div>;
    }
}

export const getBackgroundColorStatus = (status: SubprocessesStatus, classes: Record<CssRules, string>) => {
    switch (status) {
        case SubprocessesStatus.COMPLETED:
            return classes.headerCompleted;
        case SubprocessesStatus.CANCELED:
            return classes.headerCanceled;
        case SubprocessesStatus.QUEUED:
            return classes.headerQueued;
        case SubprocessesStatus.FAILED:
            return classes.headerFailed;
        case SubprocessesStatus.ACTIVE:
            return classes.headerActive;
        default:
            return classes.headerQueued;
    }
};