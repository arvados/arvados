// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { BannerAction, bannerReducerActions } from "./banner-action";

export interface BannerState {
    isOpen: boolean;
}

const initialState = {
    isOpen: false,
};

export const bannerReducer = (state: BannerState = initialState, action: BannerAction) =>
    bannerReducerActions.match(action, {
        default: () => state,
        OPEN_BANNER: () => ({
             ...state,
             isOpen: true,
        }),
        CLOSE_BANNER: () => ({
            ...state,
            isOpen: false,
       }),
    });
