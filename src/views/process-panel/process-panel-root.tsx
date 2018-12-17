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
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';

export interface ProcessPanelRootDataProps {
    process?: Process;
    subprocesses: Array<Process>;
    filters: Array<SubprocessFilterDataProps>;
    totalSubprocessesLength: number;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, process: Process) => void;
    onToggle: (status: string) => void;
    openProcessInputDialog: (uuid: string) => void;
    navigateToOutput: (uuid: string) => void;
    navigateToWorkflow: (uuid: string) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps;

export const ProcessPanelRoot = ({ process, ...props }: ProcessPanelRootProps) =>
    process
        ? <Grid container spacing={16} alignItems="stretch">
            <Grid item sm={12} md={7}>
                <ProcessInformationCard
                    process={process}
                    onContextMenu={event => props.onContextMenu(event, process)}
                    openProcessInputDialog={props.openProcessInputDialog}
                    navigateToOutput={props.navigateToOutput}
                    navigateToWorkflow={props.navigateToWorkflow}
            />
            </Grid>
            <Grid item sm={12} md={5}>
                <SubprocessesCard
                    subprocessesAmount={props.totalSubprocessesLength}
                    filters={props.filters}
                    onToggle={props.onToggle}
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

