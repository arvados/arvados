// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { navigateToRootProject } from "store/navigation/navigation-action";
import { snackbarActions } from "store/snackbar/snackbar-actions";
import { dialogActions } from "store/dialog/dialog-actions";
import { initialize } from "redux-form";

export const EXTERNAL_CREDENTIALS_PANEL = 'externalCredentialsPanel';
export const NEW_EXTERNAL_CREDENTIAL_FORM_NAME = 'newExternalCredentialFormName';

export const externalCredentialsActions = bindDataExplorerActions(EXTERNAL_CREDENTIALS_PANEL);

export type ExternalCredentialsAction = any;

export const loadExternalCredentials = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
            const user = getState().auth.user;
            if (user) {
                try {
                    dispatch(externalCredentialsActions.REQUEST_ITEMS());
                } catch (e) {
                    return;
                }
            } else {
                dispatch(navigateToRootProject);
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "You don't have permissions to view this page", hideDuration: 2000 }));
            }
        };

export const openNewExternalCredentialDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(initialize(NEW_EXTERNAL_CREDENTIAL_FORM_NAME, {}));
        dispatch(dialogActions.OPEN_DIALOG({
            id: NEW_EXTERNAL_CREDENTIAL_FORM_NAME,
            data: {},
        }));
    };

