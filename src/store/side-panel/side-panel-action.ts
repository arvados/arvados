// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { navigateTo } from 'store/navigation/navigation-action';

export const navigateFromSidePanel = (id: string) =>
    (dispatch: Dispatch) => {
        dispatch<any>(navigateTo(id));
    };
