// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from "store/store";
import { connect } from "react-redux";
import { getProcess, getSubprocesses, Process, getProcessStatus } from "store/processes/process";
import { Dispatch } from "redux";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { ProcessPanelRootDataProps, ProcessPanelRootActionProps, ProcessPanelRoot } from "./process-panel-root";
import { getProcessPanelCurrentUuid, ProcessPanel as ProcessPanelState } from "store/process-panel/process-panel";
import { groupBy } from "lodash";
import {
    loadInputs,
    loadOutputDefinitions,
    loadOutputs,
    toggleProcessPanelFilter,
    updateOutputParams,
    loadNodeJson,
} from "store/process-panel/process-panel-actions";
import { cancelRunningWorkflow, resumeOnHoldWorkflow, startWorkflow } from "store/processes/processes-actions";
import { navigateToLogCollection, pollProcessLogs, setProcessLogsPanelFilter } from "store/process-logs-panel/process-logs-panel-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { getInlineFileUrl } from "views-components/context-menu/actions/helpers";

const mapStateToProps = ({ router, auth, resources, processPanel, processLogsPanel }: RootState): ProcessPanelRootDataProps => {
    const uuid = getProcessPanelCurrentUuid(router) || "";
    const subprocesses = getSubprocesses(uuid)(resources);
    const process = getProcess(uuid)(resources);
    return {
        process,
        subprocesses: subprocesses.filter(subprocess => processPanel.filters[getProcessStatus(subprocess)]),
        filters: getFilters(processPanel, subprocesses),
        processLogsPanel: processLogsPanel,
        auth: auth,
        inputRaw: processPanel.inputRaw,
        inputParams: processPanel.inputParams,
        outputData: processPanel.outputData,
        outputDefinitions: processPanel.outputDefinitions,
        outputParams: processPanel.outputParams,
        nodeInfo: processPanel.nodeInfo,
        usageReport: (process || null) && processPanel.usageReport && getInlineFileUrl(
            `${auth.config.keepWebServiceUrl}${processPanel.usageReport.url}?api_token=${auth.apiToken}`,
            auth.config.keepWebServiceUrl,
            auth.config.keepWebInlineServiceUrl
        ),
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessPanelRootActionProps => ({
    onCopyToClipboard: (message: string) => {
        dispatch<any>(
            snackbarActions.OPEN_SNACKBAR({
                message,
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
    },
    onContextMenu: (event, process) => {
        if (process) {
            dispatch<any>(openProcessContextMenu(event, process));
        }
    },
    onToggle: status => {
        dispatch<any>(toggleProcessPanelFilter(status));
    },
    cancelProcess: uuid => dispatch<any>(cancelRunningWorkflow(uuid)),
    startProcess: uuid => dispatch<any>(startWorkflow(uuid)),
    resumeOnHoldWorkflow: uuid => dispatch<any>(resumeOnHoldWorkflow(uuid)),
    onLogFilterChange: filter => dispatch(setProcessLogsPanelFilter(filter.value)),
    navigateToLog: uuid => dispatch<any>(navigateToLogCollection(uuid)),
    loadInputs: containerRequest => dispatch<any>(loadInputs(containerRequest)),
    loadOutputs: containerRequest => dispatch<any>(loadOutputs(containerRequest)),
    loadOutputDefinitions: containerRequest => dispatch<any>(loadOutputDefinitions(containerRequest)),
    updateOutputParams: () => dispatch<any>(updateOutputParams()),
    loadNodeJson: containerRequest => dispatch<any>(loadNodeJson(containerRequest)),
    pollProcessLogs: processUuid => dispatch<any>(pollProcessLogs(processUuid)),
});

const getFilters = (processPanel: ProcessPanelState, processes: Process[]) => {
    const grouppedProcesses = groupBy(processes, getProcessStatus);
    return Object.keys(processPanel.filters).map(filter => ({
        label: filter,
        value: (grouppedProcesses[filter] || []).length,
        checked: processPanel.filters[filter],
        key: filter,
    }));
};

export const ProcessPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessPanelRoot);
