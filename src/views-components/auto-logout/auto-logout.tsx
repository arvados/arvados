// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { useIdleTimer } from "react-idle-timer";
import { Dispatch } from "redux";

import { RootState } from "~/store/store";
import { SnackbarKind, snackbarActions } from "~/store/snackbar/snackbar-actions";
import { logout } from "~/store/auth/auth-action";
import parse from "parse-duration";
import * as React from "react";
import { min } from "lodash";

interface AutoLogoutDataProps {
    sessionIdleTimeout: number;
    lastWarningDuration: number;
}

interface AutoLogoutActionProps {
    doLogout: () => void;
    doWarn: (message: string, duration: number) => void;
    doCloseWarn: () => void;
}

const mapStateToProps = (state: RootState, ownProps: any): AutoLogoutDataProps => {
    return {
        sessionIdleTimeout: parse(state.auth.config.clusterConfig.Workbench.IdleTimeout, 's') || 0,
        lastWarningDuration: ownProps.lastWarningDuration || 60,
    };
};

const mapDispatchToProps = (dispatch: Dispatch): AutoLogoutActionProps => ({
    doLogout: () => dispatch<any>(logout(true)),
    doWarn: (message: string, duration: number) =>
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message, hideDuration: duration, kind: SnackbarKind.WARNING })),
    doCloseWarn: () => dispatch(snackbarActions.CLOSE_SNACKBAR()),
});

type AutoLogoutProps = AutoLogoutDataProps & AutoLogoutActionProps;

export const AutoLogout = connect(mapStateToProps, mapDispatchToProps)(
    (props: AutoLogoutProps) => {
        let logoutTimer: NodeJS.Timer;
        const lastWarningDuration = min([props.lastWarningDuration, props.sessionIdleTimeout])! * 1000 ;

        const handleOnIdle = () => {
            logoutTimer = setTimeout(
                () => props.doLogout(), lastWarningDuration);
            props.doWarn(
                "Your session is about to be closed due to inactivity",
                lastWarningDuration);
        };

        const handleOnActive = () => {
            clearTimeout(logoutTimer);
            props.doCloseWarn();
        };

        useIdleTimer({
            timeout: (props.lastWarningDuration < props.sessionIdleTimeout)
                ? 1000 * (props.sessionIdleTimeout - props.lastWarningDuration)
                : 1,
            onIdle: handleOnIdle,
            onActive: handleOnActive,
            debounce: 500
        });

        return <span />;
    });
