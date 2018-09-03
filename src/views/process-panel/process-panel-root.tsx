// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from './process-information-card';
import { DefaultView } from '~/components/default-view/default-view';
import { ProcessIcon } from '~/components/icon/icon';
import { Process, getProcessStatus } from '~/store/processes/process';
import { SubprocessesCard } from './subprocesses-card';
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';
import { groupBy } from 'lodash';
import { ProcessSubprocesses } from '~/views/process-panel/process-subprocesses';

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
                {console.log(props.subprocesses)}
            </Grid>
            <Grid item xs={5}>
                <SubprocessesCard
                    subprocesses={props.subprocesses}
                    filters={mapGroupedProcessesToFilters(groupSubprocessesByStatus(props.subprocesses))}
                    onToggle={() => { return; }} />
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

const groupSubprocessesByStatus = (processes: Process[]) =>
    groupBy(processes, getProcessStatus);

const mapGroupedProcessesToFilters = (groupedProcesses: { [status: string]: Process[] }): SubprocessFilterDataProps[] =>
    Object
        .keys(groupedProcesses)
        .map(status => ({
            label: status,
            key: status,
            value: groupedProcesses[status].length
        }));
