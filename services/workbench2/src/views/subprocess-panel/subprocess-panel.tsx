// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { SubprocessPanelRoot, SubprocessPanelActionProps, SubprocessPanelDataProps } from "views/subprocess-panel/subprocess-panel-root";
import { RootState } from "store/store";
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { getProcess } from "store/processes/process";
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { ProcessResource } from "models/process";

const mapDispatchToProps = (dispatch: Dispatch): SubprocessPanelActionProps => ({
    onContextMenu: (event, resource, resources) => {
        const process = getProcess(resource.uuid)(resources);
        if (process) {
            dispatch<any>(openProcessContextMenu(event, process));
        }
    },
    onRowClick: ({uuid}: ProcessResource) => {
        dispatch<any>(toggleOne(uuid))
        dispatch<any>(deselectAllOthers(uuid))
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: ({uuid}: ProcessResource) => {
        dispatch<any>(navigateTo(uuid));
    },
});

const mapStateToProps = (state: RootState): Omit<SubprocessPanelDataProps,'process'> => ({
    resources: state.resources,
});

export const SubprocessPanel = connect(mapStateToProps, mapDispatchToProps)(SubprocessPanelRoot);
