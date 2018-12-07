// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { unionize, ofType, UnionOf } from "~/common/unionize";
import { RootState } from '~/store/store';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { ServiceRepository } from "~/services/services";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';
import { ApiClientAuthorization } from '~/models/api-client-authorization';

export const apiClientAuthorizationsActions = unionize({
    SET_API_CLIENT_AUTHORIZATIONS: ofType<ApiClientAuthorization[]>(),
    REMOVE_API_CLIENT_AUTHORIZATION: ofType<string>()
});

export type ApiClientAuthorizationsActions = UnionOf<typeof apiClientAuthorizationsActions>;

export const API_CLIENT_AUTHORIZATION_REMOVE_DIALOG = 'apiClientAuthorizationRemoveDialog';
export const API_CLIENT_AUTHORIZATION_ATTRIBUTES_DIALOG = 'apiClientAuthorizationAttributesDialog';
export const API_CLIENT_AUTHORIZATION_HELP_DIALOG = 'apiClientAuthorizationHelpDialog';

export const loadApiClientAuthorizationsPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            try {
                dispatch(setBreadcrumbs([{ label: 'Api client authorizations' }]));
                const response = await services.apiClientAuthorizationService.list();
                dispatch(apiClientAuthorizationsActions.SET_API_CLIENT_AUTHORIZATIONS(response.items));
            } catch (e) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
                return;
            }
        } else {
            dispatch(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
        }
    };

export const openApiClientAuthorizationAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const apiClientAuthorization = getState().apiClientAuthorizations.find(node => node.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: API_CLIENT_AUTHORIZATION_ATTRIBUTES_DIALOG, data: { apiClientAuthorization } }));
    };

export const openApiClientAuthorizationRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: API_CLIENT_AUTHORIZATION_REMOVE_DIALOG,
            data: {
                title: 'Remove api client authorization',
                text: 'Are you sure you want to remove this api client authorization?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeApiClientAuthorization = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...' }));
        try {
            await services.apiClientAuthorizationService.delete(uuid);
            dispatch(apiClientAuthorizationsActions.REMOVE_API_CLIENT_AUTHORIZATION(uuid));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Api client authorization has been successfully removed.', hideDuration: 2000 }));
        } catch (e) {
            return;
        }
    };

export const openApiClientAuthorizationsHelpDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const apiHost = getState().properties.apiHost;
        const user = getState().auth.user;
        const email = user ? user.email : '';
        const apiToken = getState().auth.apiToken;
        dispatch(dialogActions.OPEN_DIALOG({ id: API_CLIENT_AUTHORIZATION_HELP_DIALOG, data: { apiHost, apiToken, email } }));
    };