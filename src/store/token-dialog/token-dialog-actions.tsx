// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "store/dialog/dialog-actions";
import { getProperty } from 'store/properties/properties';
import { propertiesActions } from 'store/properties/properties-actions';
import { RootState } from 'store/store';

export const TOKEN_DIALOG_NAME = 'tokenDialog';
const API_HOST_PROPERTY_NAME = 'apiHost';

export interface TokenDialogData {
    token: string;
    tokenExpiration?: Date;
    apiHost: string;
    canCreateNewTokens: boolean;
}

export const setTokenDialogApiHost = (apiHost: string) =>
    propertiesActions.SET_PROPERTY({ key: API_HOST_PROPERTY_NAME, value: apiHost });

export const getTokenDialogData = (state: RootState): TokenDialogData => {
    const loginCluster = state.auth.config.clusterConfig.Login.LoginCluster;
    const canCreateNewTokens = !(loginCluster !== "" && state.auth.homeCluster !== loginCluster);

    return {
        apiHost: getProperty<string>(API_HOST_PROPERTY_NAME)(state.properties) || '',
        token: state.auth.extraApiToken || state.auth.apiToken || '',
        tokenExpiration: state.auth.extraApiToken
            ? state.auth.extraApiTokenExpiration
            : state.auth.apiTokenExpiration,
        canCreateNewTokens,
    };
};

export const openTokenDialog = dialogActions.OPEN_DIALOG({ id: TOKEN_DIALOG_NAME, data: {} });
