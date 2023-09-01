// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import copy from "copy-to-clipboard";
import { Dispatch } from "redux";
import { getNavUrl } from "routes/routes";
import { RootState } from "store/store";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export const openInNewTabAction = (resource: any) => (dispatch: Dispatch, getState: () => RootState) => {
    const url = getNavUrl(resource.uuid, getState().auth);

    if (url[0] === "/") {
        window.open(`${window.location.origin}${url}`, "_blank");
    } else if (url.length) {
        window.open(url, "_blank");
    }
};

export const copyToClipboardAction = (resources: Array<any>) => (dispatch: Dispatch, getState: () => RootState) => {
    // Copy to clipboard omits token to avoid accidental sharing

    let output = "";

    resources.forEach(resource => {
        let url = getNavUrl(resource.uuid, getState().auth, false);
        if (url[0] === "/") url = `${window.location.origin}${url}`;
        output += output.length ? `, ${url}` : url;
    });

    if (output.length) {
        const wasCopied = copy(output);
        if (wasCopied)
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: "Copied",
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                })
            );
    }
};
