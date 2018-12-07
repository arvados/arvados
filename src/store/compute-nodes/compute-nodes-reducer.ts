// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { computeNodesActions, ComputeNodesActions } from '~/store/compute-nodes/compute-nodes-actions';
import { NodeResource } from '~/models/node';

export type ComputeNodesState = NodeResource[];

const initialState: ComputeNodesState = [];

export const computeNodesReducer = (state: ComputeNodesState = initialState, action: ComputeNodesActions): ComputeNodesState =>
    computeNodesActions.match(action, {
        SET_COMPUTE_NODES: nodes => nodes,
        REMOVE_COMPUTE_NODE: (uuid: string) => state.filter((computeNode) => computeNode.uuid !== uuid),
        default: () => state
    });