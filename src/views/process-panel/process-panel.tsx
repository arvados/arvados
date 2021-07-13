// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getProcess, getSubprocesses, Process, getProcessStatus } from 'store/processes/process';
import { Dispatch } from 'redux';
import { openProcessContextMenu } from 'store/context-menu/context-menu-actions';
import { matchProcessRoute } from 'routes/routes';
import { ProcessPanelRootDataProps, ProcessPanelRootActionProps, ProcessPanelRoot } from './process-panel-root';
import { ProcessPanel as ProcessPanelState} from 'store/process-panel/process-panel';
import { groupBy } from 'lodash';
import { toggleProcessPanelFilter, navigateToOutput, openWorkflow } from 'store/process-panel/process-panel-actions';
import { openProcessInputDialog } from 'store/processes/process-input-actions';
import { cancelRunningWorkflow } from 'store/processes/processes-actions';

const mapStateToProps = ({ router, resources, processPanel }: RootState): ProcessPanelRootDataProps => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProcessRoute(pathname);
    const uuid = match ? match.params.id : '';
    const subprocesses = getSubprocesses(uuid)(resources);
    return {
        process: getProcess(uuid)(resources),
        subprocesses: subprocesses.filter(subprocess => processPanel.filters[getProcessStatus(subprocess)]),
        filters: getFilters(processPanel, subprocesses),
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ProcessPanelRootActionProps => ({
    onContextMenu: (event, process) => {
        dispatch<any>(openProcessContextMenu(event, process));
    },
    onToggle: status => {
        dispatch<any>(toggleProcessPanelFilter(status));
    },
    openProcessInputDialog: (uuid) => dispatch<any>(openProcessInputDialog(uuid)),
    navigateToOutput: (uuid) => dispatch<any>(navigateToOutput(uuid)),
    navigateToWorkflow: (uuid) => dispatch<any>(openWorkflow(uuid)),
    cancelProcess: (uuid) => dispatch<any>(cancelRunningWorkflow(uuid))
});

export const ProcessPanel = connect(mapStateToProps, mapDispatchToProps)(ProcessPanelRoot);

export const getFilters = (processPanel: ProcessPanelState, processes: Process[]) => {
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