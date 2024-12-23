// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ProcessIcon } from "components/icon/icon";
import { Process, getProcess, ProcessStatus, getProcessStatus, isProcessQueued, isProcessRunning } from "store/processes/process";
import { SubprocessPanel } from "views/subprocess-panel/subprocess-panel";
import { MPVContainer, MPVPanelContent, MPVPanelState } from "components/multi-panel-view/multi-panel-view";
import { ProcessDetailsCard } from "./process-details-card";
import { ProcessIOCard, ProcessIOCardType } from "./process-io-card";
import { ProcessResourceCard } from "./process-resource-card";
import { getProcessPanelLogs, ProcessLogsPanel } from "store/process-logs-panel/process-logs-panel";
import { ProcessLogsCard } from "./process-log-card";
import { FilterOption } from "views/process-panel/process-log-form";
import { getInputCollectionMounts } from "store/processes/processes-actions";
import { AuthState } from "store/auth/auth-reducer";
import { ProcessCmdCard } from "./process-cmd-card";
import { ContainerRequestResource } from "models/container-request";
import { ProcessPanel as ProcessPanelState } from "store/process-panel/process-panel";
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { ArvadosTheme } from 'common/custom-theme';
import { useAsyncInterval } from "common/use-async-interval";
import { WebSocketService } from "websocket/websocket-service";
import { RouteComponentProps } from 'react-router';
import { ResourcesState } from 'store/resources/resources';
import { getInlineFileUrl } from "views-components/context-menu/actions/helpers";
import { CollectionFile } from "models/collection-file";

type CssRules = "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: "100%",
    },
});

export interface ProcessPanelRootDataProps {
    resources: ResourcesState;
    processPanel: ProcessPanelState;
    processLogsPanel: ProcessLogsPanel;
    auth: AuthState;
    usageReport: CollectionFile | null;
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
    refreshProcess: (processUuid: string) => Promise<void>;
}

export type ProcessPanelRootProps = ProcessPanelRootDataProps & ProcessPanelRootActionProps & WithStyles<CssRules>;

const panelsData: MPVPanelState[] = [
    { name: "Details" },
    { name: "Logs", visible: true },
    { name: "Subprocesses" },
    { name: "Outputs" },
    { name: "Inputs" },
    { name: "Command" },
    { name: "Resources" },
];

export const ProcessPanelRoot = withStyles(styles)(({
    auth,
    resources,
    processPanel,
    processLogsPanel,
    loadInputs,
    loadOutputs,
    loadNodeJson,
    loadOutputDefinitions,
    updateOutputParams,
    pollProcessLogs,
    refreshProcess,
    onContextMenu,
    cancelProcess,
    startProcess,
    resumeOnHoldWorkflow,
    ...props
}: ProcessPanelRootProps & RouteComponentProps<{ id: string }>) => {
    const process = getProcess(props.match.params.id)(resources);
    const outputUuid = process?.containerRequest.outputUuid;
    const containerRequest = process?.containerRequest;
    const inputMounts = getInputCollectionMounts(process?.containerRequest);
    const webSocketConnected = WebSocketService.getInstance().isActive();
    const { inputRaw, inputParams, outputData, outputDefinitions, outputParams, nodeInfo, usageReport } = processPanel;

    const usageReportWithUrl = (process || null) && usageReport && getInlineFileUrl(
                `${auth.config.keepWebServiceUrl}${usageReport.url}?api_token=${auth.apiToken}`,
                auth.config.keepWebServiceUrl,
                auth.config.keepWebInlineServiceUrl
            )

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

    const maxHeight = "100%";

    // Trigger processing output params when raw or definitions change
    React.useEffect(() => {
        updateOutputParams();
    }, [outputData, outputDefinitions, updateOutputParams]);

    // If WebSocket not connected, poll queued/running process for status updates
    const shouldPoll =
        !webSocketConnected &&
        process && (
            isProcessQueued(process)
            || isProcessRunning(process)
            // Status is unknown if has containerUuid but container resource not loaded
            || getProcessStatus(process) === ProcessStatus.UNKNOWN
        );
    useAsyncInterval(async () => {
        process && await refreshProcess(process.containerRequest.uuid);
    }, shouldPoll ? 15000 : null);

        return process ? (
            <MPVContainer
                className={props.classes.root}
                spacing={1}
                panelStates={panelsData}
                justifyContent="flex-start"
                direction="column"
                wrap="nowrap">
                <MPVPanelContent
                    forwardProps
                    item
                    xs="auto"
                    data-cy="process-details">
                    <ProcessDetailsCard
                        process={process}
                        onContextMenu={event => onContextMenu(event, process)}
                        cancelProcess={cancelProcess}
                        startProcess={startProcess}
                        resumeOnHoldWorkflow={resumeOnHoldWorkflow}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    item
                    xs
                    minHeight={maxHeight}
                    maxHeight={maxHeight}
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
                        pollProcessLogs={pollProcessLogs}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    item
                    maxHeight={maxHeight}
                    data-cy="process-children">
                    <SubprocessPanel process={process} />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    item
                    maxHeight={maxHeight}
                    data-cy="process-outputs">
                    <ProcessIOCard
                        label={ProcessIOCardType.OUTPUT}
                        process={process}
                        params={outputParams}
                        raw={outputData?.raw}
                        failedToLoadOutputCollection={outputData?.failedToLoadOutputCollection}
                        outputUuid={outputUuid || ""}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    item
                    maxHeight={maxHeight}
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
                    xs="auto"
                    item
                    maxHeight={"50%"}
                    data-cy="process-cmd">
                    <ProcessCmdCard
                        onCopy={props.onCopyToClipboard}
                        process={process}
                    />
                </MPVPanelContent>
                <MPVPanelContent
                    forwardProps
                    xs
                    item
                    data-cy="process-resources">
                    <ProcessResourceCard
                        process={process}
                        nodeInfo={nodeInfo}
                        usageReport={usageReportWithUrl}
                    />
                </MPVPanelContent>
            </MPVContainer>
        ) : (
            <NotFoundView
                icon={ProcessIcon}
                messages={["Process not found"]}
            />
        );
}
);
