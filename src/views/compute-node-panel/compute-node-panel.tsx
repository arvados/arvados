// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { } from '~/store/compute-nodes/compute-nodes-actions';
import {
    ComputeNodePanelRoot,
    ComputeNodePanelRootDataProps,
    ComputeNodePanelRootActionProps
} from '~/views/compute-node-panel/compute-node-panel-root';
import { openComputeNodeContextMenu } from '~/store/context-menu/context-menu-actions';

const mapStateToProps = (state: RootState): ComputeNodePanelRootDataProps => {
    return {
        computeNodes: state.computeNodes,
        hasComputeNodes: state.computeNodes.length > 0
    };
};

const mapDispatchToProps = (dispatch: Dispatch): ComputeNodePanelRootActionProps => ({
    openRowOptions: (event, computeNode) => {
        dispatch<any>(openComputeNodeContextMenu(event, computeNode));
    }
});

export const ComputeNodePanel = connect(mapStateToProps, mapDispatchToProps)(ComputeNodePanelRoot);