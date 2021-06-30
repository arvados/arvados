// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { useIdleTimer } from "react-idle-timer";
import { Dispatch } from "redux";

import { RootState } from "store/store";
import { SnackbarKind, snackbarActions } from "store/snackbar/snackbar-actions";
import { logout } from "store/auth/auth-action";
import parse from "parse-duration";
import React from "react";
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

const mapStateToProps = (state: RootState, ownProps: any): AutoLogoutDataProps => ({
    sessionIdleTimeout: parse(state.auth.config.clusterConfig.Workbench.IdleTimeout, 's') || 0,
    lastWarningDuration: ownProps.lastWarningDuration || 60,
});

const mapDispatchToProps = (dispatch: Dispatch): AutoLogoutActionProps => ({
    doLogout: () => dispatch<any>(logout(true)),
    doWarn: (message: string, duration: number) =>
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message, hideDuration: duration, kind: SnackbarKind.WARNING })),
    doCloseWarn: () => dispatch(snackbarActions.CLOSE_SNACKBAR()),
});

export type AutoLogoutProps = AutoLogoutDataProps & AutoLogoutActionProps;

const debounce = (delay: number | undefined, fn: Function) => {
    let timerId: number | null;
    return (...args: any[]) => {
        if (timerId) { clearTimeout(timerId); }
        timerId = setTimeout(() => {
            fn(...args);
            timerId = null;
        }, delay);
    };
};

export const LAST_ACTIVE_TIMESTAMP = 'lastActiveTimestamp';

export const AutoLogoutComponent = (props: AutoLogoutProps) => {
    let logoutTimer: NodeJS.Timer;
    const lastWarningDuration = min([props.lastWarningDuration, props.sessionIdleTimeout])! * 1000;

    // Runs once after render
    React.useEffect(() => {
        window.addEventListener('storage', handleStorageEvents);
        // Component cleanup
        return () => {
            window.removeEventListener('storage', handleStorageEvents);
        };
    }, []);

    const handleStorageEvents = (e: StorageEvent) => {
        if (e.key === LAST_ACTIVE_TIMESTAMP) {
            // Other tab activity detected by a localStorage change event.
            debounce(500, () => {
                handleOnActive();
                reset();
            })();
        }
    };

    const handleOnIdle = () => {
        logoutTimer = setTimeout(
            () => props.doLogout(), lastWarningDuration);
        props.doWarn(
            "Your session is about to be closed due to inactivity",
            lastWarningDuration);
    };

    const handleOnActive = () => {
        if (logoutTimer) { clearTimeout(logoutTimer); }
        props.doCloseWarn();
    };

    const handleOnAction = () => {
        // Notify the other tabs there was some activity.
        const now = (new Date).getTime();
        localStorage.setItem(LAST_ACTIVE_TIMESTAMP, now.toString());
    };

    const { reset } = useIdleTimer({
        timeout: (props.lastWarningDuration < props.sessionIdleTimeout)
            ? 1000 * (props.sessionIdleTimeout - props.lastWarningDuration)
            : 1,
        onIdle: handleOnIdle,
        onActive: handleOnActive,
        onAction: handleOnAction,
        debounce: 500
    });

    return <span />;
};

export const AutoLogout = connect(mapStateToProps, mapDispatchToProps)(AutoLogoutComponent);
