// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getProcess, getSubprocesses, Process, getProcessStatus } from 'store/processes/process';
import { Dispatch } from 'redux';
import { openProcessContextMenu } from 'store/context-menu/context-menu-actions';
import {
    ProcessPanelRootDataProps,
    ProcessPanelRootActionProps,
    ProcessPanelRoot
} from './process-panel-root';
import {
    getProcessPanelCurrentUuid,
    ProcessPanel as ProcessPanelState
} from 'store/process-panel/process-panel';
import { groupBy } from 'lodash';
import {
    loadInputs,
    loadOutputDefinitions,
    loadOutputs,
    toggleProcessPanelFilter,
    updateOutputParams,
    loadNodeJson
} from 'store/process-panel/process-panel-actions';
import { cancelRunningWorkflow, startWorkflow } from 'store/processes/processes-actions';
import { navigateToLogCollection, setProcessLogsPanelFilter } from 'store/process-logs-panel/process-logs-panel-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

const mapStateToProps = ({ router, auth, resources, processPanel, processLogsPanel }: RootState): ProcessPanelRootDataProps => {
    const uuid = getProcessPanelCurrentUuid(router) || '';
    const subprocesses = getSubprocesses(uuid)(resources);
    return {
        process: getProcess(uuid)(resources),
        subprocesses: subprocesses.filter(subprocess => processPanel.filters[getProcessStatus(subprocess)]),
        filters: getFilters(processPanel, subprocesses),
        processLogsPanel: processLogsPanel,
        auth: auth,
        inputRaw: processPanel.inputRaw,
        inputParams: processPanel.inputParams,
        outputRaw: processPanel.outputRaw,
        outputDefinitions: processPanel.outputDefinitions,
        outputParams: processPanel.outputParams,
        nodeInfo: processPanel.nodeInfo,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessPanelRootActionProps => ({
    onCopyToClipboard: (message: string) => {
        dispatch<any>(snackbarActions.OPEN_SNACKBAR({
            message,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS,
        }));
    },
    onContextMenu: (event, process) => {
        dispatch<any>(openProcessContextMenu(event, process));
    },
    onToggle: status => {
        dispatch<any>(toggleProcessPanelFilter(status));
    },
    cancelProcess: (uuid) => dispatch<any>(cancelRunningWorkflow(uuid)),
    startProcess: (uuid) => dispatch<any>(startWorkflow(uuid)),
    onLogFilterChange: (filter) => dispatch(setProcessLogsPanelFilter(filter.value)),
    navigateToLog: (uuid) => dispatch<any>(navigateToLogCollection(uuid)),
    loadInputs: (containerRequest) => dispatch<any>(loadInputs(containerRequest)),
    loadOutputs: (containerRequest) => dispatch<any>(loadOutputs(containerRequest)),
    loadOutputDefinitions: (containerRequest) => dispatch<any>(loadOutputDefinitions(containerRequest)),
    updateOutputParams: () => dispatch<any>(updateOutputParams()),
    loadNodeJson: (containerRequest) => dispatch<any>(loadNodeJson(containerRequest)),
});

const getFilters = (processPanel: ProcessPanelState, processes: Process[]) => {
    const grouppedProcesses = groupBy(processes, getProcessStatus);
    return Object
        .keys(processPanel.filters)
        .map(filter => ({
            label: filter,
            value: (grouppedProcesses[filter] || []).length,
            checked: processPanel.filters[filter],
            key: filter,
        }));
};

export const ProcessPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessPanelRoot);
