// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as copy from 'copy-to-clipboard';
import { ResourceKind } from 'models/resource';
import { getClipboardUrl } from 'views-components/context-menu/actions/helpers';

const getUrl = (resource: any) => {
    let url = null;
    const { uuid, kind } = resource;

    if (kind === ResourceKind.COLLECTION) {
        url = `/collections/${uuid}`;
    }
    if (kind === ResourceKind.PROJECT) {
        url = `/projects/${uuid}`;
    }

    return url;
};

export const openInNewTabAction = (resource: any) => () => {
    const url = getUrl(resource);

    if (url) {
        window.open(`${window.location.origin}${url}`, '_blank');
    }
};

export const copyToClipboardAction = (resource: any) => () => {
    const url = getUrl(resource);

    if (url) {
        copy(getClipboardUrl(url, false));
    }
};