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
    toggleProcessPanelFilter,
} from 'store/process-panel/process-panel-actions';
import { cancelRunningWorkflow } from 'store/processes/processes-actions';
import { navigateToLogCollection, setProcessLogsPanelFilter } from 'store/process-logs-panel/process-logs-panel-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

const mapStateToProps = ({ router, resources, processPanel, processLogsPanel }: RootState): ProcessPanelRootDataProps => {
    const uuid = getProcessPanelCurrentUuid(router) || '';
    const subprocesses = getSubprocesses(uuid)(resources);
    return {
        process: getProcess(uuid)(resources),
        subprocesses: subprocesses.filter(subprocess => processPanel.filters[getProcessStatus(subprocess)]),
        filters: getFilters(processPanel, subprocesses),
        processLogsPanel: processLogsPanel,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessPanelRootActionProps => ({
    onLogCopyToClipboard: (message: string) => {
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
    onLogFilterChange: (filter) => dispatch(setProcessLogsPanelFilter(filter.value)),
    navigateToLog: (uuid) => dispatch<any>(navigateToLogCollection(uuid)),
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
