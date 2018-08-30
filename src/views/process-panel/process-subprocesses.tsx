// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessSubprocessesCard } from '~/views/process-panel/process-subprocesses-card';

export enum SubprocessesStatus {
    ACTIVE = 'Active',
    COMPLETED = 'Completed',
    QUEUED = 'Queued',
    FAILED = 'Failed',
    CANCELED = 'Canceled'
}

export interface ProcessSubprocessesDataProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

interface SubprocessesProps {
    title: string;
    status: string;
    runtime?: string;
}

export const ProcessSubprocesses = ({ onContextMenu }: ProcessSubprocessesDataProps) => {
    return <Grid container spacing={16}>
        {items.map(it =>
            <Grid item xs={2} key={it.title}>
                <ProcessSubprocessesCard onContextMenu={onContextMenu} items={it} />
            </Grid>
        )}
    </Grid>;
};

const items: Array<SubprocessesProps> = [
    {
        title: 'cos1',
        status: SubprocessesStatus.ACTIVE
    },
    {
        title: 'cos2',
        status: SubprocessesStatus.FAILED
    },
    {
        title: 'cos3',
        status: SubprocessesStatus.QUEUED
    },
    {
        title: 'cos4',
        status: SubprocessesStatus.CANCELED
    },
    {
        title: 'cos5',
        status: SubprocessesStatus.COMPLETED
    },
    {
        title: 'cos6',
        status: SubprocessesStatus.COMPLETED
    },
    {
        title: 'cos7',
        status: SubprocessesStatus.COMPLETED
    },
];

