// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import {
    ComputeNodePanelRoot,
    ComputeNodePanelRootDataProps,
    ComputeNodePanelRootActionProps
} from 'views/compute-node-panel/compute-node-panel-root';
import { openComputeNodeContextMenu } from 'store/context-menu/context-menu-actions';

const mapStateToProps = (state: RootState): ComputeNodePanelRootDataProps => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ComputeNodePanelRootActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        dispatch<any>(openComputeNodeContextMenu(event, resourceUuid));
    },
    onItemClick: (resourceUuid: string) => { return; },
    onItemDoubleClick: uuid => { return; }
});

export const ComputeNodePanel = connect(mapStateToProps, mapDispatchToProps)(ComputeNodePanelRoot);