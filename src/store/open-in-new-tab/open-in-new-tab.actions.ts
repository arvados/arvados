// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { ResourceKind } from '~/models/resource';
import { unionize, ofType } from '~/common/unionize';

export const openInNewTabActions = unionize({
    COPY_STORE: ofType<{}>(),
    OPEN_COLLECTION_IN_NEW_TAB: ofType<string>(),
    OPEN_PROJECT_IN_NEW_TAB: ofType<string>()
});

export const openInNewTabAction = (resource: any) => (dispatch: Dispatch) => {
    const { uuid, kind } = resource;

    dispatch(openInNewTabActions.COPY_STORE());

    if (kind === ResourceKind.COLLECTION) {
        dispatch(openInNewTabActions.OPEN_COLLECTION_IN_NEW_TAB(uuid));
    } else if (kind === ResourceKind.PROJECT) {
        dispatch(openInNewTabActions.OPEN_PROJECT_IN_NEW_TAB(uuid));
    }
};