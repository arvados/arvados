// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { navigateTo } from 'store/navigation/navigation-action';
import { WorkflowPanelView } from 'views/workflow-panel/workflow-panel-view';
import { WorfklowPanelActionProps, WorkflowPanelDataProps } from './workflow-panel-view';
import { showWorkflowDetails, getWorkflowDetails } from 'store/workflow-panel/workflow-panel-actions';
import { RootState } from 'store/store';
import { WorkflowResource } from 'models/workflow';

const mapStateToProps = (state: RootState): WorkflowPanelDataProps => ({
    workflow: getWorkflowDetails(state)
});

const mapDispatchToProps = (dispatch: Dispatch): WorfklowPanelActionProps => ({
    handleRowDoubleClick: ({uuid}: WorkflowResource) => {
        dispatch<any>(navigateTo(uuid));
    },

    handleRowClick: ({uuid}: WorkflowResource) => {
        dispatch(showWorkflowDetails(uuid));
    }
});

export const WorkflowPanel = connect(mapStateToProps, mapDispatchToProps)(WorkflowPanelView);
