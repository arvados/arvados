// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "~/store/dialog/dialog-actions";
import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import {snackbarActions, SnackbarKind} from "~/store/snackbar/snackbar-actions";
import { FormErrors, reset, startSubmit, stopSubmit } from "redux-form";
import { KeyType } from "~/models/ssh-key";
import {
    AuthorizedKeysServiceError,
    getAuthorizedKeysServiceError
} from "~/services/authorized-keys-service/authorized-keys-service";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import {
    authActions,
} from "~/store/auth/auth-action";

export const SSH_KEY_CREATE_FORM_NAME = 'sshKeyCreateFormName';
export const SSH_KEY_PUBLIC_KEY_DIALOG = 'sshKeyPublicKeyDialog';
export const SSH_KEY_REMOVE_DIALOG = 'sshKeyRemoveDialog';
export const SSH_KEY_ATTRIBUTES_DIALOG = 'sshKeyAttributesDialog';

export interface SshKeyCreateFormDialogData {
    publicKey: string;
    name: string;
}

export const openSshKeyCreateDialog = () => dialogActions.OPEN_DIALOG({ id: SSH_KEY_CREATE_FORM_NAME, data: {} });

export const openPublicKeyDialog = (name: string, publicKey: string) =>
    dialogActions.OPEN_DIALOG({ id: SSH_KEY_PUBLIC_KEY_DIALOG, data: { name, publicKey } });

export const openSshKeyAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const sshKey = getState().auth.sshKeys.find(it => it.uuid === uuid);
        dispatch(dialogActions.OPEN_DIALOG({ id: SSH_KEY_ATTRIBUTES_DIALOG, data: { sshKey } }));
    };

export const openSshKeyRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: SSH_KEY_REMOVE_DIALOG,
            data: {
                title: 'Remove public key',
                text: 'Are you sure you want to remove this public key?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeSshKey = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.authorizedKeysService.delete(uuid);
        dispatch(authActions.REMOVE_SSH_KEY(uuid));
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Public Key has been successfully removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
    };

export const createSshKey = (data: SshKeyCreateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getState().auth.user!.uuid;
        const { name, publicKey } = data;
        dispatch(startSubmit(SSH_KEY_CREATE_FORM_NAME));
        try {
            const newSshKey = await services.authorizedKeysService.create({
                name,
                publicKey,
                keyType: KeyType.SSH,
                authorizedUserUuid: userUuid
            });
            dispatch(authActions.ADD_SSH_KEY(newSshKey));
            dispatch(dialogActions.CLOSE_DIALOG({ id: SSH_KEY_CREATE_FORM_NAME }));
            dispatch(reset(SSH_KEY_CREATE_FORM_NAME));
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Public key has been successfully created.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
        } catch (e) {
            const error = getAuthorizedKeysServiceError(e);
            if (error === AuthorizedKeysServiceError.UNIQUE_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key already exists.' } as FormErrors));
            } else if (error === AuthorizedKeysServiceError.INVALID_PUBLIC_KEY) {
                dispatch(stopSubmit(SSH_KEY_CREATE_FORM_NAME, { publicKey: 'Public key is invalid' } as FormErrors));
            }
        }
    };

export const loadSshKeysPanel = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'SSH Keys'}]));
            const response = await services.authorizedKeysService.list();
            dispatch(authActions.SET_SSH_KEYS(response.items));
        } catch (e) {
            return;
        }
    };

