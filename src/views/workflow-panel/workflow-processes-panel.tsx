// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { WorkflowProcessesPanelRoot, WorkflowProcessesPanelActionProps, WorkflowProcessesPanelDataProps } from "views/workflow-panel/workflow-processes-panel-root";
import { RootState } from "store/store";
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { getProcess } from "store/processes/process";

const mapDispatchToProps = (dispatch: Dispatch): WorkflowProcessesPanelActionProps => ({
    onContextMenu: (event, resourceUuid, resources) => {
        const process = getProcess(resourceUuid)(resources);
        if (process) {
            dispatch<any>(openProcessContextMenu(event, process));
        }
    },
    onItemClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    },
});

const mapStateToProps = (state: RootState): WorkflowProcessesPanelDataProps => ({
    resources: state.resources,
});

export const WorkflowProcessesPanel = connect(mapStateToProps, mapDispatchToProps)(WorkflowProcessesPanelRoot);
