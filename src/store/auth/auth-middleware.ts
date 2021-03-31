// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Middleware } from "redux";
import { authActions, } from "./auth-action";
import { ServiceRepository, setAuthorizationHeader, removeAuthorizationHeader } from "~/services/services";
import { initSessions } from "~/store/auth/auth-action-session";
import { User } from "~/models/user";
import { RootState } from '~/store/store';
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { WORKBENCH_LOADING_SCREEN } from '~/store/workbench/workbench-actions';

export const authMiddleware = (services: ServiceRepository): Middleware => store => next => action => {
    // Middleware to update external state (local storage, window
    // title) to ensure that they stay in sync with redux state.

    authActions.match(action, {
        INIT_USER: ({ user, token }) => {
            // The "next" method passes the action to the next
            // middleware in the chain, or the reducer.  That means
            // after next() returns, the action has (presumably) been
            // applied by the reducer to update the state.
            next(action);

            const state: RootState = store.getState();

            if (state.auth.apiToken) {
                services.authService.saveApiToken(state.auth.apiToken);
                setAuthorizationHeader(services, state.auth.apiToken);
            } else {
                services.authService.removeApiToken();
                services.authService.removeSessions();
                removeAuthorizationHeader(services);
            }

            store.dispatch<any>(initSessions(services.authService, state.auth.remoteHostsConfig[state.auth.localCluster], user));
            if (!user.isActive) {
                // As a special case, if the user is inactive, they
                // may be able to self-activate using the "activate"
                // method.  Note, for this to work there can't be any
                // unsigned user agreements, we assume the API server is just going to
                // rubber-stamp our activation request.  At some point in the future we'll
                // want to either add support for displaying/signing user
                // agreements or get rid of self-activation.
                // For more details, see:
                // https://doc.arvados.org/master/admin/user-management.html

                store.dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
                services.userService.activate(user.uuid).then((user: User) => {
                    store.dispatch(authActions.INIT_USER({ user, token }));
                    store.dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
                }).catch(() => {
                    store.dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
                });
            }
        },
        SET_CONFIG: ({ config }) => {
            document.title = `Arvados Workbench (${config.uuidPrefix})`;
            next(action);
        },
        LOGOUT: ({ deleteLinkData }) => {
            next(action);
            if (deleteLinkData) {
                services.linkAccountService.removeAccountToLink();
            }
            const token = services.authService.getApiToken();
            services.authService.removeApiToken();
            services.authService.removeSessions();
            services.authService.removeUser();
            removeAuthorizationHeader(services);
            services.authService.logout(token || '');
        },
        default: () => next(action)
    });
};
