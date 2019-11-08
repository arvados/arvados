// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
/* import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { ProcessResource } from '~/models/process';
import { getResource } from '~/store/resources/resources';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { REMOVE_PROCESS_DIALOG } from '~/store/processes/processes-actions';
*/
export const SUBPROCESS_PANEL_ID = "subprocessPanel";
export const SUBPROCESS_ATTRIBUTES_DIALOG = 'subprocessAttributesDialog';
export const subprocessPanelActions = bindDataExplorerActions(SUBPROCESS_PANEL_ID);

export const loadSubprocessPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(subprocessPanelActions.REQUEST_ITEMS());
    };

/*
export const openSubprocessAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const subprocess = getResource<ProcessResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: SUBPROCESS_ATTRIBUTES_DIALOG, data: { subprocess } }));
    };

export const openLinkRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: REMOVE_PROCESS_DIALOG,
            data: {
                title: 'Remove process',
                text: 'Are you sure you want to remove this process?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeSubprocess = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        try {
            await services.containerRequestService.delete(uuid);
            dispatch(subprocessPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process has been successfully removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            return;
        }
    };*/