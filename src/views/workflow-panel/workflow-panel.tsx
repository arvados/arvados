// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { navigateTo } from '~/store/navigation/navigation-action';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { WorkflowPanelView } from '~/views/workflow-panel/workflow-panel-view';

const mapDispatchToProps = (dispatch: Dispatch) => ({

    handleRowDoubleClick: (uuid: string) => {
        dispatch<any>(navigateTo(uuid));
    },
    
    handleRowClick: (uuid: string) => {
        dispatch(loadDetailsPanel(uuid));
    }
});

export const WorkflowPanel= connect(undefined, mapDispatchToProps)(
    (props) => <WorkflowPanelView {...props}/>);