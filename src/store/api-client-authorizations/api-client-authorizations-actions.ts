// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { ServiceRepository } from "~/services/services";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';
import { ApiClientAuthorization } from '~/models/api-client-authorization';
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { getResource } from '~/store/resources/resources';


export const API_CLIENT_AUTHORIZATION_PANEL_ID = 'apiClientAuthorizationPanelId';
export const apiClientAuthorizationsActions = bindDataExplorerActions(API_CLIENT_AUTHORIZATION_PANEL_ID);

export const API_CLIENT_AUTHORIZATION_REMOVE_DIALOG = 'apiClientAuthorizationRemoveDialog';
export const API_CLIENT_AUTHORIZATION_ATTRIBUTES_DIALOG = 'apiClientAuthorizationAttributesDialog';
export const API_CLIENT_AUTHORIZATION_HELP_DIALOG = 'apiClientAuthorizationHelpDialog';


export const loadApiClientAuthorizationsPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user;
        if (user && user.isAdmin) {
            try {
                dispatch(setBreadcrumbs([{ label: 'Api client authorizations' }]));
                dispatch(apiClientAuthorizationsActions.REQUEST_ITEMS());
            } catch (e) {
                return;
            }
        } else {
            dispatch(navigateToRootProject);
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
        }
    };

export const openApiClientAuthorizationAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const apiClientAuthorization = getResource<ApiClientAuthorization>(uuid)(resources);
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
            dispatch(apiClientAuthorizationsActions.REQUEST_ITEMS());
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