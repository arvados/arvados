// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { getNavUrl } from "routes/routes";
import { RootState } from "store/store";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

const dispatchCopyResult = (dispatch: Dispatch, text: string) => {
    if (text) {
        navigator.clipboard.writeText(text).then(() => {
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: "Copied",
                    hideDuration: 8000,
                    kind: SnackbarKind.SUCCESS,
                })
            );
        }).catch(() => {
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: "Failed to copy",
                    hideDuration: 10000,
                    kind: SnackbarKind.ERROR,
                })
            );
        });
    } else {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Failed to copy",
                hideDuration: 10000,
                kind: SnackbarKind.ERROR,
            })
        );
    }
};

export const copyLinkToClipboardAction = (resources: Array<any>) => (dispatch: Dispatch, getState: () => RootState) => {
    // Copy link to clipboard omits token to avoid accidental sharing

    let url = getNavUrl(resources[0].uuid, getState().auth, false);

    let textToCopy = "";
    if (url[0] === "/") textToCopy = `${window.location.origin}${url}`;
    else if (url.length) {
        textToCopy = url;
    }

    dispatchCopyResult(dispatch, textToCopy);
};

export const copyStringToClipboardAction = (text: string) => (dispatch: Dispatch, getState: () => RootState) => {
    dispatchCopyResult(dispatch, text);
};
