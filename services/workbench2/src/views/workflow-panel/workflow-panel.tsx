// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { navigateTo } from 'store/navigation/navigation-action';
import { WorkflowPanelView } from 'views/workflow-panel/workflow-panel-view';
import { WorfklowPanelActionProps, WorkflowPanelDataProps } from './workflow-panel-view';
import { showWorkflowDetails } from 'store/workflow-panel/workflow-panel-actions';
import { RootState } from 'store/store';
import { WORKFLOW_PANEL_DETAILS_UUID } from 'store/workflow-panel/workflow-panel-actions';

const mapStateToProps = (state: RootState): WorkflowPanelDataProps => {
    const uuid = state.properties[WORKFLOW_PANEL_DETAILS_UUID];
    const workflows = state.runProcessPanel.workflows;
    return {
        uuid,
        workflows,
    }
};

const mapDispatchToProps = (dispatch: Dispatch): WorfklowPanelActionProps => ({
    handleRowDoubleClick: (uuid: string) => {
        dispatch<any>(navigateTo(uuid));
    },

    handleRowClick: (uuid: string) => {
        dispatch(showWorkflowDetails(uuid));
    }
});

export const WorkflowPanel = connect(mapStateToProps, mapDispatchToProps)(WorkflowPanelView);
