// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid, StyleRulesCallback, WithStyles, withStyles } from "@material-ui/core";
import { DefaultView } from "components/default-view/default-view";
import { ProcessIcon } from "components/icon/icon";
import { Process } from "store/processes/process";
import { SubprocessPanel } from "views/subprocess-panel/subprocess-panel";
import { SubprocessFilterDataProps } from "components/subprocess-filter/subprocess-filter";
import { MPVContainer, MPVPanelContent, MPVPanelState } from "components/multi-panel-view/multi-panel-view";
import { ArvadosTheme } from "common/custom-theme";
import { ProcessDetailsCard } from "./process-details-card";
import { ProcessIOCard, ProcessIOCardType, ProcessIOParameter } from "./process-io-card";
import { ProcessResourceCard } from "./process-resource-card";
import { getProcessPanelLogs, ProcessLogsPanel } from "store/process-logs-panel/process-logs-panel";
import { ProcessLogsCard } from "./process-log-card";
import { FilterOption } from "views/process-panel/process-log-form";
import { getInputCollectionMounts } from "store/processes/processes-actions";
import { WorkflowInputsData } from "models/workflow";
import { CommandOutputParameter } from "cwlts/mappings/v1.0/CommandOutputParameter";
import { AuthState } from "store/auth/auth-reducer";
import { ProcessCmdCard } from "./process-cmd-card";
import { ContainerRequestResource } from "models/container-request";
import { OutputDetails, NodeInstanceType } from "store/process-panel/process-panel";

type CssRules = "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: "100%",
    },
});

export interface ProcessPanelRootDataProps {
    process?: Process;
    subprocesses: Array<Process>;
    filters: Array<SubprocessFilterDataProps>;
    processLogsPanel: ProcessLogsPanel;
    auth: AuthState;
    inputRaw: WorkflowInputsData | null;
    inputParams: ProcessIOParameter[] | null;
    outputRaw: OutputDetails | null;
    outputDefinitions: CommandOutputParameter[];
    outputParams: ProcessIOParameter[] | null;
    nodeInfo: NodeInstanceType | null;
}

export interface ProcessPanelRootActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, process: Process) => void;
    onToggle: (status: string) => void;
    cancelProcess: (uuid: string) => void;
    startProcess: (uuid: string) => void;
    resumeOnHoldWorkflow: (uuid: string) => void;
    onLogFilterChange: (filter: FilterOption) => void;
    navigateToLog: (uuid: string) => void;
    onCopyToClipboard: (uuid: string) => void;
    loadInputs: (containerRequest: ContainerRequestResource) => void;
    loadOutputs: (containerRequest: ContainerRequestResource) => void;
    loadNodeJson: (containerRequest: ContainerRequestResource) => void;
    loadOutputDefinitions: (containerRequest: ContainerRequestResource) => void;
    updateOutputParams: () => void;
    pollProcessLogs: (processUuid: string) => Promise<void>;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps & WithStyles<CssRules>;

const panelsData: MPVPanelState[] = [
    { name: "Details" },
    { name: "Command" },
    { name: "Logs", visible: true },
    { name: "Inputs" },
    { name: "Outputs" },
    { name: "Resources" },
    { name: "Subprocesses" },
];

export const ProcessPanelRoot = withStyles(styles)(
    ({
        process,
        auth,
        processLogsPanel,
        inputRaw,
        inputParams,
        outputRaw,
        outputDefinitions,
        outputParams,
        nodeInfo,
        loadInputs,
        loadOutputs,
        loadNodeJson,
        loadOutputDefinitions,
        updateOutputParams,
        ...props
    }: ProcessPanelRootProps) => {
        const outputUuid = process?.containerRequest.outputUuid;
        const containerRequest = process?.containerRequest;
        const inputMounts = getInputCollectionMounts(process?.containerRequest);

        React.useEffect(() => {
            if (containerRequest) {
                // Load inputs from mounts or props
                loadInputs(containerRequest);
                // Fetch raw output (loads from props or keep)
                loadOutputs(containerRequest);
                // Loads output definitions from mounts into store
                loadOutputDefinitions(containerRequest);
                // load the assigned instance type from node.json in
                // the log collection
                loadNodeJson(containerRequest);
            }
        }, [containerRequest, loadInputs, loadOutputs, loadOutputDefinitions, loadNodeJson]);

        // Trigger processing output params when raw or definitions change
        React.useEffect(() => {
            updateOutputParams();
        }, [outputRaw, outputDefinitions, updateOutputParams]);

        return process ? (
            <MPVContainer
                className={props.classes.root}
                spacing={8}
                panelStates={panelsData}
                justify-content="flex-start"
                direction="column"
                wrap="nowrap">
                <MPVPanelContent
                    forwardProps
                    xs="auto"
                    data-cy="process-details">
                    <ProcessDetailsCard
                        process={process}
                        onContextMenu={event => props.onContextMenu(event, process)}
                        cancelProcess={props.cancelProcess}
                        startProcess={props.startProcess}
                        resumeOnHoldWorkflow={props.resumeOnHoldWorkflow}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs="auto"
                    data-cy="process-cmd">
                    <ProcessCmdCard
                        onCopy={props.onCopyToClipboard}
                        process={process}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    minHeight="50%"
                    data-cy="process-logs">
                    <ProcessLogsCard
                        onCopy={props.onCopyToClipboard}
                        process={process}
                        lines={getProcessPanelLogs(processLogsPanel)}
                        selectedFilter={{
                            label: processLogsPanel.selectedFilter,
                            value: processLogsPanel.selectedFilter,
                        }}
                        filters={processLogsPanel.filters.map(filter => ({ label: filter, value: filter }))}
                        onLogFilterChange={props.onLogFilterChange}
                        navigateToLog={props.navigateToLog}
                        pollProcessLogs={props.pollProcessLogs}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    maxHeight="50%"
                    data-cy="process-inputs">
                    <ProcessIOCard
                        label={ProcessIOCardType.INPUT}
                        process={process}
                        params={inputParams}
                        raw={inputRaw}
                        mounts={inputMounts}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    maxHeight="50%"
                    data-cy="process-outputs">
                    <ProcessIOCard
                        label={ProcessIOCardType.OUTPUT}
                        process={process}
                        params={outputParams}
                        raw={outputRaw?.rawOutputs}
                        outputUuid={outputUuid || ""}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    data-cy="process-resources">
                    <ProcessResourceCard
                        process={process}
                        nodeInfo={nodeInfo}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    maxHeight="50%"
                    data-cy="process-children">
                    <SubprocessPanel />
                </MPVPanelContent>
            </MPVContainer>
        ) : (
            <Grid
                container
                alignItems="center"
                justify="center"
                style={{ minHeight: "100%" }}>
                <DefaultView
                    icon={ProcessIcon}
                    messages={["Process not found"]}
                />
            </Grid>
        );
    }
);
