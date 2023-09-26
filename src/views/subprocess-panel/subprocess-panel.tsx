// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { openProcessContextMenu, openRunningProcessContextMenu } from "store/context-menu/context-menu-actions";
import { SubprocessPanelRoot, SubprocessPanelActionProps, SubprocessPanelDataProps } from "views/subprocess-panel/subprocess-panel-root";
import { RootState } from "store/store";
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { getProcess } from "store/processes/process";
import { isProcessCancelable } from "store/processes/process";

const mapDispatchToProps = (dispatch: Dispatch): SubprocessPanelActionProps => ({
    onContextMenu: (event, resourceUuid, resources) => {
        const process = getProcess(resourceUuid)(resources);
        if (process && isProcessCancelable(process)) {
            dispatch<any>(openRunningProcessContextMenu(event, process));
        } else if (process) {
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

const mapStateToProps = (state: RootState): SubprocessPanelDataProps => ({
    resources: state.resources,
});

export const SubprocessPanel = connect(mapStateToProps, mapDispatchToProps)(SubprocessPanelRoot);
