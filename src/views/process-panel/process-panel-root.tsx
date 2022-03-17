// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Grid, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { ProcessInformationCard } from './process-information-card';
import { DefaultView } from 'components/default-view/default-view';
import { ProcessIcon } from 'components/icon/icon';
import { Process } from 'store/processes/process';
import { SubprocessPanel } from 'views/subprocess-panel/subprocess-panel';
import { SubprocessFilterDataProps } from 'components/subprocess-filter/subprocess-filter';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { ArvadosTheme } from 'common/custom-theme';
import { ProcessDetailsCard } from './process-details-card';
import { getProcessPanelLogs, ProcessLogsPanel } from 'store/process-logs-panel/process-logs-panel';
import { ProcessLogsCard } from './process-log-card';
import { FilterOption } from 'views/process-panel/process-log-form';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
});

export interface ProcessPanelRootDataProps {
    process?: Process;
    subprocesses: Array<Process>;
    filters: Array<SubprocessFilterDataProps>;
    processLogsPanel: ProcessLogsPanel;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, process: Process) => void;
    onToggle: (status: string) => void;
    openProcessInputDialog: (uuid: string) => void;
    navigateToOutput: (uuid: string) => void;
    navigateToWorkflow: (uuid: string) => void;
    cancelProcess: (uuid: string) => void;
    onLogFilterChange: (filter: FilterOption) => void;
    navigateToLog: (uuid: string) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps & WithStyles<CssRules>;

const panelsData: MPVPanelState[] = [
    {name: "Info"},
    {name: "Details", visible: false},
    {name: "Logs", visible: true},
    {name: "Subprocesses"},
];

export const ProcessPanelRoot = withStyles(styles)(
    ({ process, processLogsPanel, ...props }: ProcessPanelRootProps) =>
    process
        ? <MPVContainer className={props.classes.root} spacing={8} panelStates={panelsData}  justify-content="flex-start" direction="column" wrap="nowrap">
            <MPVPanelContent forwardProps xs="auto">
                <ProcessInformationCard
                    process={process}
                    onContextMenu={event => props.onContextMenu(event, process)}
                    openProcessInputDialog={props.openProcessInputDialog}
                    navigateToOutput={props.navigateToOutput}
                    openWorkflow={props.navigateToWorkflow}
                    cancelProcess={props.cancelProcess}
                />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs="auto">
                <ProcessDetailsCard process={process} />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs>
                <ProcessLogsCard
                    process={process}
                    lines={getProcessPanelLogs(processLogsPanel)}
                    selectedFilter={{
                        label: processLogsPanel.selectedFilter,
                        value: processLogsPanel.selectedFilter
                    }}
                    filters={processLogsPanel.filters.map(
                        filter => ({ label: filter, value: filter })
                    )}
                    onLogFilterChange={props.onLogFilterChange}
                    navigateToLog={props.navigateToLog}
                />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs>
                <SubprocessPanel />
            </MPVPanelContent>
        </MPVContainer>
        : <Grid container
            alignItems='center'
            justify='center'
            style={{ minHeight: '100%' }}>
            <DefaultView
                icon={ProcessIcon}
                messages={['Process not found']} />
        </Grid>);
