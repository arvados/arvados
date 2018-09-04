// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessSubprocessesCard } from '~/views/process-panel/process-subprocesses-card';
import { Process } from '~/store/processes/process';

export interface ProcessSubprocessesDataProps {
    subprocesses: Array<Process>;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

export const ProcessSubprocesses = ({ onContextMenu, subprocesses }: ProcessSubprocessesDataProps) => {
    return <Grid container spacing={16}>
        {subprocesses.map(subprocess =>
            <Grid item xs={12} sm={6} md={4} lg={2} key={subprocess.containerRequest.uuid}>
                <ProcessSubprocessesCard onContextMenu={onContextMenu} subprocess={subprocess} />
            </Grid>
        )}
    </Grid>;
};
