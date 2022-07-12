// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState } from 'react';
import { Grid, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DefaultView } from 'components/default-view/default-view';
import { ProcessIcon } from 'components/icon/icon';
import { Process } from 'store/processes/process';
import { SubprocessPanel } from 'views/subprocess-panel/subprocess-panel';
import { SubprocessFilterDataProps } from 'components/subprocess-filter/subprocess-filter';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { ArvadosTheme } from 'common/custom-theme';
import { ProcessDetailsCard } from './process-details-card';
import { getInputDisplayValue, ProcessIOCard, ProcessIOParameter } from './process-io-card';

import { getProcessPanelLogs, ProcessLogsPanel } from 'store/process-logs-panel/process-logs-panel';
import { ProcessLogsCard } from './process-log-card';
import { FilterOption } from 'views/process-panel/process-log-form';
import { getInputs } from 'store/processes/processes-actions';
import { CommandInputParameter, getInputId } from 'models/workflow';
import { CommandOutputParameter } from 'cwlts/mappings/v1.0/CommandOutputParameter';
import { AuthState } from 'store/auth/auth-reducer';

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
    auth: AuthState;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, process: Process) => void;
    onToggle: (status: string) => void;
    cancelProcess: (uuid: string) => void;
    onLogFilterChange: (filter: FilterOption) => void;
    navigateToLog: (uuid: string) => void;
    onLogCopyToClipboard: (uuid: string) => void;
    fetchOutputs: (uuid: string, fetchOutputs) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps & WithStyles<CssRules>;

type OutputDetails = {
    rawOutputs?: any;
    pdh?: string;
}

const panelsData: MPVPanelState[] = [
    {name: "Details"},
    {name: "Logs", visible: true},
    {name: "Inputs"},
    {name: "Outputs"},
    {name: "Subprocesses"},
];

export const ProcessPanelRoot = withStyles(styles)(
    ({ process, auth, processLogsPanel, fetchOutputs, ...props }: ProcessPanelRootProps) => {

    const [outputDetails, setOutputs] = useState<OutputDetails>({});
    const [rawInputs, setInputs] = useState<CommandInputParameter[]>([]);


    const [processedOutputs, setProcessedOutputs] = useState<ProcessIOParameter[]>([]);
    const [processedInputs, setProcessedInputs] = useState<ProcessIOParameter[]>([]);

    const outputUuid = process?.containerRequest.outputUuid;
    const requestUuid = process?.containerRequest.uuid;

    React.useEffect(() => {
        if (outputUuid) {
            fetchOutputs(outputUuid, setOutputs);
        }
    }, [outputUuid, fetchOutputs]);

    React.useEffect(() => {
        if (outputDetails.rawOutputs) {
            setProcessedOutputs(formatOutputData(outputDetails.rawOutputs, outputDetails.pdh, auth));
        }
    }, [outputDetails, auth]);

    React.useEffect(() => {
        if (process) {
            const rawInputs = getInputs(process.containerRequest);
            setInputs(rawInputs);
            setProcessedInputs(formatInputData(rawInputs, auth));
        }
    }, [requestUuid, auth, process]);

    return process
        ? <MPVContainer className={props.classes.root} spacing={8} panelStates={panelsData}  justify-content="flex-start" direction="column" wrap="nowrap">
            <MPVPanelContent forwardProps xs="auto" data-cy="process-details">
                <ProcessDetailsCard
                    process={process}
                    onContextMenu={event => props.onContextMenu(event, process)}
                    cancelProcess={props.cancelProcess}
                />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs maxHeight='50%' data-cy="process-logs">
                <ProcessLogsCard
                    onCopy={props.onLogCopyToClipboard}
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
            <MPVPanelContent forwardProps xs="auto" data-cy="process-inputs">
                <ProcessIOCard
                    label="Inputs"
                    params={processedInputs}
                    raw={rawInputs}
                 />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs="auto" data-cy="process-outputs">
                <ProcessIOCard
                    label="Outputs"
                    params={processedOutputs}
                    raw={outputDetails.rawOutputs}
                 />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs maxHeight='50%' data-cy="process-children">
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
        </Grid>;
    }
);

const formatInputData = (inputs: CommandInputParameter[], auth: AuthState): ProcessIOParameter[] => {
    return inputs.map(input => {
        return {
            id: getInputId(input),
            doc: input.label || "",
            value: getInputDisplayValue(auth, input)
        };
    });
};

const formatOutputData = (rawData: any, pdh: string | undefined, auth: AuthState): ProcessIOParameter[] => {
    if (!rawData) { return []; }
    return Object.keys(rawData).map((id): ProcessIOParameter => {
        const multiple = rawData[id].length > 0;
        const outputArray = multiple ? rawData[id] : [rawData[id]];
        return {
            id,
            doc: outputArray.map((outputParam: CommandOutputParameter) => (outputParam.doc))
                        // Doc can be string or string[], concat conveniently works with both
                        .reduce((acc: string[], input: string | string[]) => (acc.concat(input)), [])
                        // Remove undefined and empty doc strings
                        .filter(str => str)
                        .join(", "),
            value: outputArray.map(outputParam => getInputDisplayValue(auth, {
                    type: outputParam.class,
                    value: outputParam,
                    ...outputParam
                }, pdh, outputParam.secondaryFiles))
                .reduce((acc: ProcessIOParameter[], params: ProcessIOParameter[]) => (acc.concat(params)), [])
        };
    });
};
