// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import copy from 'copy-to-clipboard';
import { Dispatch } from 'redux';
import { getNavUrl } from 'routes/routes';
import { RootState } from 'store/store';

export const openInNewTabAction = (resource: any) => (dispatch: Dispatch, getState: () => RootState) => {
    const url = getNavUrl(resource.uuid, getState().auth);

    if (url[0] === '/') {
        window.open(`${window.location.origin}${url}`, '_blank');
    } else if (url.length) {
        window.open(url, '_blank');
    }
};

export const copyToClipboardAction = (resource: any) => (dispatch: Dispatch, getState: () => RootState) => {
    // Copy to clipboard omits token to avoid accidental sharing
    const url = getNavUrl(resource.uuid, getState().auth, false);

    if (url[0] === '/') {
        copy(`${window.location.origin}${url}`);
    } else if (url.length) {
        copy(url);
    }
};
