// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { dialogActions } from "store/dialog/dialog-actions";
import { CreateExternalCredentialFormDialogData, UpdateExternalCredentialFormDialogData } from "store/external-credentials/external-credential-dialog-data";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { getCheckedListUuids } from "store/multiselect/multiselect-actions";
import { FormErrors, initialize, reset, startSubmit, stopSubmit } from "redux-form";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { getResource } from "store/resources/resources";
import { ExternalCredential } from "models/external-credential";
import { showGroupedCommonResourceResultSnackbars } from "store/resources/resources-actions";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export const EXTERNAL_CREDENTIALS_PANEL = 'externalCredentialsPanel';
export const CREATE_EXTERNAL_CREDENTIAL_FORM_NAME = 'newExternalCredentialFormName';
export const REMOVE_EXTERNAL_CREDENTIAL_DIALOG = "removeExternalCredentialDialog";
export const UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME = "updateExternalCredentialFormName";

export const externalCredentialsActions = bindDataExplorerActions(EXTERNAL_CREDENTIALS_PANEL);

export const loadExternalCredentials = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(progressIndicatorActions.START_WORKING(EXTERNAL_CREDENTIALS_PANEL));
            try {
                dispatch(externalCredentialsActions.REQUEST_ITEMS());
            } catch (e) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: e.message,
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            } finally {
                dispatch(progressIndicatorActions.STOP_WORKING(EXTERNAL_CREDENTIALS_PANEL));
            }
        };

export const openNewExternalCredentialDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(initialize(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME, {}));
        dispatch(dialogActions.OPEN_DIALOG({
            id: CREATE_EXTERNAL_CREDENTIAL_FORM_NAME,
            data: {},
        }));
    };

export const createExternalCredential = (data: CreateExternalCredentialFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME));
        try {
            dispatch(progressIndicatorActions.START_WORKING(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME));
            const newExternalCredential = await services.externalCredentialsService.create(data);
            dispatch(externalCredentialsActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: CREATE_EXTERNAL_CREDENTIAL_FORM_NAME }));
            dispatch(reset(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME));
            return newExternalCredential;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME, { name: "Credential with the same name already exists." } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: CREATE_EXTERNAL_CREDENTIAL_FORM_NAME }));
                const errMsg = e.errors ? e.errors.join("") : "Could not create the credential";
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: errMsg,
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
            return;
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME));
        }
    };

export const openRemoveExternalCredentialDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const numOfCredentials = getCheckedListUuids(getState()).length;
        dispatch(
            dialogActions.OPEN_DIALOG({
                id: REMOVE_EXTERNAL_CREDENTIAL_DIALOG,
                data: {
                    title: "Remove External Credentials",
                    text: numOfCredentials === 1 ? "Are you sure you want to remove this credential?" : `Are you sure you want to remove these ${numOfCredentials} credentials?`,
                    confirmButtonLabel: "Remove",
                    uuid: resource.uuid,
                    resource,
                },
            })
        );
    };

export const removeExternalCredentialPermanently = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(progressIndicatorActions.START_WORKING(EXTERNAL_CREDENTIALS_PANEL));
        const credentialsToRemove = getCheckedListUuids(getState());

        const messageFuncMap = {
            [CommonResourceServiceError.NONE]: (count: number) => count > 1 ? `Removed ${count} items` : `Item removed`,
            [CommonResourceServiceError.PERMISSION_ERROR_FORBIDDEN]: (count: number) => count > 1 ? `Remove ${count} items failed: Access Denied` : `Remove failed: Access Denied`,
            [CommonResourceServiceError.UNKNOWN]: (count: number) => count > 1 ? `Remove ${count} items failed` : `Remove failed`,
        };

        await Promise.allSettled(credentialsToRemove.map(credential => services.externalCredentialsService.delete(credential))).then((promises) => {
            const { success } = showGroupedCommonResourceResultSnackbars(dispatch, promises, messageFuncMap);
            if (success.length) {
                dispatch<any>(loadExternalCredentials());
            }
        });

        dispatch(progressIndicatorActions.STOP_WORKING(EXTERNAL_CREDENTIALS_PANEL));
    };

export const openExternalCredentialUpdateDialog = (resource: ContextMenuResource) => (dispatch: Dispatch, getState: () => RootState) => {
    const credential = getResource<ExternalCredential>(resource.uuid)(getState().resources);
    dispatch(initialize(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME, credential));
    dispatch(
        dialogActions.OPEN_DIALOG({id: UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME, data: {}})
    );
};

export const updateExternalCredential =
    (credential: UpdateExternalCredentialFormDialogData & { uuid?: string }) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = credential.uuid || "";
        dispatch(startSubmit(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME));
        try {
            dispatch(progressIndicatorActions.START_WORKING(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME));
            const updatedCredential = await services.externalCredentialsService.update(
                uuid,
                {
                    name: credential.name,
                    description: credential.description,
                    credentialClass: credential.credentialClass,
                    externalId: credential.externalId,
                    expiresAt: credential.expiresAt,
                    // if no secret, don't include the field at all
                    ...(credential.secret && credential.secret.length > 0 ? { secret: credential.secret } : {}),
                    scopes: credential.scopes || [],
                },
                false
            );
            dispatch(externalCredentialsActions.REQUEST_ITEMS());
            dispatch(reset(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME));
            dispatch(dialogActions.CLOSE_DIALOG({ id: UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME }));
            return updatedCredential;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME, { name: "Credential with the same name already exists." } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME }));
                const errMsg = e.errors ? e.errors.join("") : "There was an error while updating the credential";
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: errMsg,
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
            return;
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME));
        }
    };
