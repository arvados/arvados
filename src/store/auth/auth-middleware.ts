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
    authActions.match(action, {
        INIT: ({ user, token }) => {
            next(action);
            const state: RootState = store.getState();

            if (state.auth.apiToken) {
                services.authService.saveApiToken(state.auth.apiToken);
                setAuthorizationHeader(services, state.auth.apiToken);
            } else {
                services.authService.removeApiToken();
                removeAuthorizationHeader(services);
            }

            store.dispatch<any>(initSessions(services.authService, state.auth.remoteHostsConfig[state.auth.localCluster], user));
            if (!user.isActive) {
                store.dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
                services.userService.activate(user.uuid).then((user: User) => {
                    store.dispatch(authActions.INIT({ user, token }));
                    store.dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
                }).catch(() => {
                    store.dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
                });
            }
        },
        CONFIG: ({ config }) => {
            document.title = `Arvados Workbench (${config.uuidPrefix})`;
            next(action);
        },
        LOGOUT: ({ deleteLinkData }) => {
            next(action);
            if (deleteLinkData) {
                services.linkAccountService.removeAccountToLink();
            }
            services.authService.removeApiToken();
            services.authService.removeUser();
            removeAuthorizationHeader(services);
            services.authService.logout();
        },
        default: () => next(action)
    });
};
