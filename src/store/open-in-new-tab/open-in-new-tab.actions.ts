// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import copy from 'copy-to-clipboard';
import { getResourceUrl } from 'routes/routes';
import { getClipboardUrl } from 'views-components/context-menu/actions/helpers';

export const openInNewTabAction = (resource: any) => () => {
    const url = getResourceUrl(resource.uuid);

    if (url) {
        window.open(`${window.location.origin}${url}`, '_blank');
    }
};

export const copyToClipboardAction = (resource: any) => () => {
    const url = getResourceUrl(resource.uuid);

    if (url) {
        copy(getClipboardUrl(url, false));
    }
};
