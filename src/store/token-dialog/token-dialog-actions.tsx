// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "~/store/dialog/dialog-actions";
import { getProperty } from '~/store/properties/properties';
import { propertiesActions } from '~/store/properties/properties-actions';
import { RootState } from '~/store/store';

export const TOKEN_DIALOG_NAME = 'tokenDialog';
const API_HOST_PROPERTY_NAME = 'apiHost';

export interface TokenDialogData {
    token: string;
    apiHost: string;
}

export const setTokenDialogApiHost = (apiHost: string) =>
    propertiesActions.SET_PROPERTY({ key: API_HOST_PROPERTY_NAME, value: apiHost });

export const getTokenDialogData = (state: RootState): TokenDialogData => ({
    apiHost: getProperty<string>(API_HOST_PROPERTY_NAME)(state.properties) || '',
    token: state.auth.apiToken || '',
});

export const openTokenDialog = dialogActions.OPEN_DIALOG({ id: TOKEN_DIALOG_NAME, data: {} });
