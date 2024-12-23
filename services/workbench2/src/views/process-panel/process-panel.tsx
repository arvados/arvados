// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from "store/store";
import { connect } from "react-redux";
import { Dispatch } from "redux";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { ProcessPanelRootDataProps, ProcessPanelRootActionProps, ProcessPanelRoot } from "./process-panel-root";
import {
    loadInputs,
    loadOutputDefinitions,
    loadOutputs,
    toggleProcessPanelFilter,
    updateOutputParams,
    loadNodeJson,
    loadProcess,
} from "store/process-panel/process-panel-actions";
import { cancelRunningWorkflow, resumeOnHoldWorkflow, startWorkflow } from "store/processes/processes-actions";
import { navigateToLogCollection, pollProcessLogs, setProcessLogsPanelFilter } from "store/process-logs-panel/process-logs-panel-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

const mapStateToProps = ({ auth, resources, processPanel, processLogsPanel }: RootState): ProcessPanelRootDataProps => {
    return {
        resources,
        processLogsPanel: processLogsPanel,
        auth: auth,
        processPanel: processPanel,
        usageReport: processPanel.usageReport,
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
    refreshProcess: processUuid => dispatch<any>(loadProcess(processUuid)),
});

export const ProcessPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessPanelRoot);
