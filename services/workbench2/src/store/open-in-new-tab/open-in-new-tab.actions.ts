// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { getNavUrl } from "routes/routes";
import { RootState } from "store/store";

export const openInNewTabAction = (resource: any) => (dispatch: Dispatch, getState: () => RootState) => {
    const url = getNavUrl(resource.uuid, getState().auth);

    if (url[0] === "/") {
        window.open(`${window.location.origin}${url}`, "_blank", "noopener");
    } else if (url.length) {
        window.open(url, "_blank", "noopener");
    }
};
