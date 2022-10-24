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
import { getIOParamDisplayValue, ProcessIOCard, ProcessIOCardType, ProcessIOParameter } from './process-io-card';

import { getProcessPanelLogs, ProcessLogsPanel } from 'store/process-logs-panel/process-logs-panel';
import { ProcessLogsCard } from './process-log-card';
import { FilterOption } from 'views/process-panel/process-log-form';
import { getInputs, getInputCollectionMounts, getOutputParameters, getRawInputs } from 'store/processes/processes-actions';
import { CommandInputParameter, getIOParamId } from 'models/workflow';
import { CommandOutputParameter } from 'cwlts/mappings/v1.0/CommandOutputParameter';
import { AuthState } from 'store/auth/auth-reducer';
import { ProcessCmdCard } from './process-cmd-card';
import { ContainerRequestResource } from 'models/container-request';

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
    onCopyToClipboard: (uuid: string) => void;
    fetchOutputs: (containerRequest: ContainerRequestResource, fetchOutputs) => void;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps & WithStyles<CssRules>;

type OutputDetails = {
    rawOutputs?: any;
    pdh?: string;
}

const panelsData: MPVPanelState[] = [
    {name: "Details"},
    {name: "Command"},
    {name: "Logs", visible: true},
    {name: "Inputs"},
    {name: "Outputs"},
    {name: "Subprocesses"},
];

export const ProcessPanelRoot = withStyles(styles)(
    ({ process, auth, processLogsPanel, fetchOutputs, ...props }: ProcessPanelRootProps) => {

    const [outputDetails, setOutputs] = useState<OutputDetails | undefined>(undefined);
    const [outputDefinitions, setOutputDefinitions] = useState<CommandOutputParameter[]>([]);
    const [rawInputs, setInputs] = useState<CommandInputParameter[] | undefined>(undefined);

    const [processedOutputs, setProcessedOutputs] = useState<ProcessIOParameter[] | undefined>(undefined);
    const [processedInputs, setProcessedInputs] = useState<ProcessIOParameter[] | undefined>(undefined);

    const outputUuid = process?.containerRequest.outputUuid;
    const requestUuid = process?.containerRequest.uuid;

    const containerRequest = process?.containerRequest;

    const inputMounts = getInputCollectionMounts(process?.containerRequest);

    // Resets state when changing processes
    React.useEffect(() => {
        setOutputs(undefined);
        setOutputDefinitions([]);
        setInputs(undefined);
        setProcessedOutputs(undefined);
        setProcessedInputs(undefined);
    }, [requestUuid]);

    // Fetch raw output (async for fetching from keep)
    React.useEffect(() => {
        if (containerRequest) {
            fetchOutputs(containerRequest, setOutputs);
        }
    }, [containerRequest, fetchOutputs]);

    // Format raw output into ProcessIOParameter[] when it changes
    React.useEffect(() => {
        if (outputDetails !== undefined && outputDetails.rawOutputs && containerRequest) {
            const newOutputDefinitions = getOutputParameters(containerRequest);
            // Avoid setting output definitions back to [] when mounts briefly go missing
            if (newOutputDefinitions.length) {
                setOutputDefinitions(newOutputDefinitions);
            }
            setProcessedOutputs(formatOutputData(outputDefinitions, outputDetails.rawOutputs, outputDetails.pdh, auth));
        }
    }, [outputDetails, auth, containerRequest, outputDefinitions]);

    // Fetch raw inputs and format into ProcessIOParameter[]
    //   Can be sync because inputs are either already in containerRequest mounts or props
    React.useEffect(() => {
        if (containerRequest) {
            // Since mounts can disappear and reappear, only set inputs if raw / processed inputs is undefined or new inputs has content
            const newRawInputs = getRawInputs(containerRequest);
            if (rawInputs === undefined || (newRawInputs && newRawInputs.length)) {
                setInputs(newRawInputs);
            }
            const newInputs = getInputs(containerRequest);
            if (processedInputs === undefined || (newInputs && newInputs.length)) {
                setProcessedInputs(formatInputData(newInputs, auth));
            }
        }
    }, [requestUuid, auth, containerRequest, processedInputs, rawInputs]);

    return process
        ? <MPVContainer className={props.classes.root} spacing={8} panelStates={panelsData}  justify-content="flex-start" direction="column" wrap="nowrap">
            <MPVPanelContent forwardProps xs="auto" data-cy="process-details">
                <ProcessDetailsCard
                    process={process}
                    onContextMenu={event => props.onContextMenu(event, process)}
                    cancelProcess={props.cancelProcess}
                />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs="auto" data-cy="process-cmd">
                <ProcessCmdCard
                    onCopy={props.onCopyToClipboard}
                    process={process} />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs maxHeight='50%' data-cy="process-logs">
                <ProcessLogsCard
                    onCopy={props.onCopyToClipboard}
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
            <MPVPanelContent forwardProps xs maxHeight='50%' data-cy="process-inputs">
                <ProcessIOCard
                    label={ProcessIOCardType.INPUT}
                    process={process}
                    params={processedInputs}
                    raw={rawInputs}
                    mounts={inputMounts}
                 />
            </MPVPanelContent>
            <MPVPanelContent forwardProps xs maxHeight='50%' data-cy="process-outputs">
                <ProcessIOCard
                    label={ProcessIOCardType.OUTPUT}
                    process={process}
                    params={processedOutputs}
                    raw={outputDetails?.rawOutputs}
                    outputUuid={outputUuid || ""}
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
            id: getIOParamId(input),
            label: input.label || "",
            value: getIOParamDisplayValue(auth, input)
        };
    });
};

const formatOutputData = (definitions: CommandOutputParameter[], values: any, pdh: string | undefined, auth: AuthState): ProcessIOParameter[] => {
    return definitions.map(output => {
        return {
            id: getIOParamId(output),
            label: output.label || "",
            value: getIOParamDisplayValue(auth, Object.assign(output, { value: values[getIOParamId(output)] || [] }), pdh)
        };
    });
};
