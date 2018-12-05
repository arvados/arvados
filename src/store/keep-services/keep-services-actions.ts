// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { RootState } from '~/store/store';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { ServiceRepository } from "~/services/services";
import { KeepServiceResource } from '~/models/keep-services';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';

export const keepServicesActions = unionize({
    SET_KEEP_SERVICES: ofType<KeepServiceResource[]>(),
    REMOVE_KEEP_SERVICE: ofType<string>()
});

export type KeepServicesActions = UnionOf<typeof keepServicesActions>;

export const KEEP_SERVICE_REMOVE_DIALOG = 'keepServiceRemoveDialog';
export const KEEP_SERVICE_ATTRIBUTES_DIALOG = 'keepServiceAttributesDialog';

export const loadKeepServicesPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if(user && user.isAdmin) {
            try {
                dispatch(setBreadcrumbs([{ label: 'Keep Services' }]));
                const response = await services.keepService.list();
                dispatch(keepServicesActions.SET_KEEP_SERVICES(response.items));
            } catch (e) {
                return;
            }
        } else {
            dispatch(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
        }
    };

export const openKeepServiceAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const keepService = getState().keepServices.find(it => it.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: KEEP_SERVICE_ATTRIBUTES_DIALOG, data: { keepService } }));
    };

export const openKeepServiceRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: KEEP_SERVICE_REMOVE_DIALOG,
            data: {
                title: 'Remove keep service',
                text: 'Are you sure you want to remove this keep service?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeKeepService = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
        try {
            await services.keepService.delete(uuid);
            dispatch(keepServicesActions.REMOVE_KEEP_SERVICE(uuid));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Keep service has been successfully removed.', hideDuration: 2000 }));
        } catch (e) {
            return;
        }
    };