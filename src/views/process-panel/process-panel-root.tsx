// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from './process-information-card';
import { DefaultView } from '~/components/default-view/default-view';
import { ProcessIcon } from '~/components/icon/icon';
import { Process } from '~/store/processes/process';
import { SubprocessesCard } from './subprocesses-card';
import { ProcessSubprocesses } from '~/views/process-panel/process-subprocesses';
import { SubprocessesStatus } from '~/views/process-panel/process-subprocesses-card';

type CssRules = 'headerActive' | 'headerCompleted' | 'headerQueued' | 'headerFailed' | 'headerCanceled';

export interface ProcessPanelRootDataProps {
    process?: Process;
    subprocesses: Array<Process>;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps;

export const ProcessPanelRoot = (props: ProcessPanelRootProps) =>
    props.process
        ? <Grid container spacing={16}>
            <Grid item xs={7}>
                <ProcessInformationCard
                    process={props.process}
                    onContextMenu={props.onContextMenu} />
            </Grid>
            <Grid item xs={5}>
                <SubprocessesCard
                    subprocesses={4}
                    filters={[
                        {
                            key: 'queued',
                            value: 1,
                            label: 'Queued',
                            checked: true
                        }, {
                            key: 'active',
                            value: 2,
                            label: 'Active',
                            checked: true
                        },
                        {
                            key: 'completed',
                            value: 2,
                            label: 'Completed',
                            checked: true
                        },
                        {
                            key: 'failed',
                            value: 2,
                            label: 'Failed',
                            checked: true
                        }
                    ]}
                    onToggle={() => { return; }}
                />
            </Grid>
            <Grid item xs={12}>
                <ProcessSubprocesses
                    subprocesses={props.subprocesses}
                    onContextMenu={props.onContextMenu} />
            </Grid>
        </Grid>
        : <Grid container
            alignItems='center'
            justify='center'
            style={{ minHeight: '100%' }}>
            <DefaultView
                icon={ProcessIcon}
                messages={['Process not found']} />
        </Grid>;
