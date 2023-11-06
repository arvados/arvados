// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { appInfoActions, AppInfoAction } from "store/app-info/app-info-actions";

export interface AppInfoState {
    buildInfo: string;
}

const initialState = {
    buildInfo: ''
};

export const appInfoReducer = (state: AppInfoState = initialState, action: AppInfoAction) =>
    appInfoActions.match(action, {
        SET_BUILD_INFO: buildInfo => ({ ...state, buildInfo }),
        default: () => state
    });
