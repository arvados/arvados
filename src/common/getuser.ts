// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';

export const getUserUuid = (state: RootState) => {
    const user = state.auth.user;
    if (user) {
        return user.uuid;
    } else {
        return undefined;
    }
};
