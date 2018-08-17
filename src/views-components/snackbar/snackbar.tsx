// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { connect } from "react-redux";
import { RootState } from "~/store/store";
import MaterialSnackbar, { SnackbarProps } from "@material-ui/core/Snackbar";
import { Dispatch } from "redux";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";

const mapStateToProps = (state: RootState): SnackbarProps => ({
    anchorOrigin: { vertical: "bottom", horizontal: "center" },
    open: state.snackbar.open,
    message: <span>{state.snackbar.message}</span>,
    autoHideDuration: state.snackbar.hideDuration
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<SnackbarProps, "onClose"> => ({
    onClose: (event: any, reason: string) => {
        if (reason !== "clickaway") {
            dispatch(snackbarActions.CLOSE_SNACKBAR());
        }
    }
});

export const Snackbar = connect(mapStateToProps, mapDispatchToProps)(MaterialSnackbar);
