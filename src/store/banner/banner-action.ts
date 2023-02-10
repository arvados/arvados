// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { unionize, UnionOf } from 'common/unionize';

export const bannerReducerActions = unionize({
    OPEN_BANNER: {},
    CLOSE_BANNER: {},
});

export type BannerAction = UnionOf<typeof bannerReducerActions>;

export const openBanner = () =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(bannerReducerActions.OPEN_BANNER());
    };

export const closeBanner = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState) => {
        dispatch(bannerReducerActions.CLOSE_BANNER());
    };

export default {
    openBanner,
    closeBanner
};
