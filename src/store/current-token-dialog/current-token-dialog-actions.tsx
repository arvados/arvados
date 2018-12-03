// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dialogActions } from "~/store/dialog/dialog-actions";
import { getProperty } from '~/store/properties/properties';
import { propertiesActions } from '~/store/properties/properties-actions';
import { RootState } from '~/store/store';

export const CURRENT_TOKEN_DIALOG_NAME = 'currentTokenDialog';
const API_HOST_PROPERTY_NAME = 'apiHost';

export interface CurrentTokenDialogData {
    currentToken: string;
    apiHost: string;
}

export const setCurrentTokenDialogApiHost = (apiHost: string) =>
    propertiesActions.SET_PROPERTY({ key: API_HOST_PROPERTY_NAME, value: apiHost });

export const getCurrentTokenDialogData = (state: RootState): CurrentTokenDialogData => ({
    apiHost: getProperty<string>(API_HOST_PROPERTY_NAME)(state.properties) || '',
    currentToken: state.auth.apiToken || '',
});

export const openCurrentTokenDialog = dialogActions.OPEN_DIALOG({ id: CURRENT_TOKEN_DIALOG_NAME, data: {} });
