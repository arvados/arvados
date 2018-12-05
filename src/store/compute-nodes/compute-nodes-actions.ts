// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { RootState } from '~/store/store';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { ServiceRepository } from "~/services/services";
import { NodeResource } from '~/models/node';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';

export const computeNodesActions = unionize({
    SET_COMPUTE_NODES: ofType<NodeResource[]>(),
    REMOVE_COMPUTE_NODE: ofType<string>()
});

export type ComputeNodesActions = UnionOf<typeof computeNodesActions>;

export const COMPUTE_NODE_REMOVE_DIALOG = 'computeNodeRemoveDialog';
export const COMPUTE_NODE_ATTRIBUTES_DIALOG = 'computeNodeAttributesDialog';

export const loadComputeNodesPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            try {
                dispatch(setBreadcrumbs([{ label: 'Compute Nodes' }]));
                const response = await services.nodeService.list();
                dispatch(computeNodesActions.SET_COMPUTE_NODES(response.items));
            } catch (e) {
                return;
            }
        } else {
            dispatch(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
        }
    };

export const openComputeNodeAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const computeNode = getState().computeNodes.find(node => node.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: COMPUTE_NODE_ATTRIBUTES_DIALOG, data: { computeNode } }));
    };

export const openComputeNodeRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: COMPUTE_NODE_REMOVE_DIALOG,
            data: {
                title: 'Remove compute node',
                text: 'Are you sure you want to remove this compute node?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeComputeNode = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
        try {
            await services.nodeService.delete(uuid);
            dispatch(computeNodesActions.REMOVE_COMPUTE_NODE(uuid));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Compute node has been successfully removed.', hideDuration: 2000 }));
        } catch (e) {
            return;
        }
    };