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
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { ContainerRequestResource } from "models/container-request";

const mapDispatchToProps = (dispatch: Dispatch): WorkflowProcessesPanelActionProps => ({
    onContextMenu: (event, resource) => {
        dispatch<any>(openProcessContextMenu(event, resource));
    },
    onItemClick: (resource: ContainerRequestResource) => {
        dispatch<any>(toggleOne(resource.uuid))
        dispatch<any>(deselectAllOthers(resource.uuid))
        dispatch<any>(loadDetailsPanel(resource.uuid));
    },
    onItemDoubleClick: ({uuid}: ContainerRequestResource) => {
        dispatch<any>(navigateTo(uuid));
    },
});

const mapStateToProps = (state: RootState): WorkflowProcessesPanelDataProps => ({
    resources: state.resources,
});

export const WorkflowProcessesPanel = connect(mapStateToProps, mapDispatchToProps)(WorkflowProcessesPanelRoot);
