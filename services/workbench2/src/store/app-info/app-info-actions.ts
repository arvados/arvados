// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from 'common/unionize';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { getBuildInfo } from 'common/app-info';

export const appInfoActions = unionize({
    SET_BUILD_INFO: ofType<string>()
});

export type AppInfoAction = UnionOf<typeof appInfoActions>;

export const setBuildInfo = () => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) =>
        dispatch(appInfoActions.SET_BUILD_INFO(getBuildInfo()));



