// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from 'store/store';
import { setBreadcrumbs } from 'store/breadcrumbs/breadcrumbs-actions';
import { dialogActions } from 'store/dialog/dialog-actions';
import {snackbarActions, SnackbarKind} from 'store/snackbar/snackbar-actions';
import { navigateToRootProject } from 'store/navigation/navigation-action';
import { bindDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { getResource } from 'store/resources/resources';
import { ServiceRepository } from "services/services";
import { NodeResource } from 'models/node';

export const COMPUTE_NODE_PANEL_ID = "computeNodeId";
export const computeNodesActions = bindDataExplorerActions(COMPUTE_NODE_PANEL_ID);

export const COMPUTE_NODE_REMOVE_DIALOG = 'computeNodeRemoveDialog';
export const COMPUTE_NODE_ATTRIBUTES_DIALOG = 'computeNodeAttributesDialog';

export const loadComputeNodesPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            try {
                dispatch(setBreadcrumbs([{ label: 'Compute Nodes' }]));
                dispatch(computeNodesActions.REQUEST_ITEMS());
            } catch (e) {
                return;
            }
        } else {
            dispatch(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const openComputeNodeAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const computeNode = getResource<NodeResource>(uuid)(resources);
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
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        try {
            await services.nodeService.delete(uuid);
            dispatch(computeNodesActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Compute node has been successfully removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            return;
        }
    };