// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import Axios from "axios";
import { getUserFullname, User } from "~/models/user";
import { authActions } from "~/store/auth/auth-action";
import { Config, DISCOVERY_URL } from "~/common/config";
import { Session, SessionStatus } from "~/models/session";
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { AuthService, UserDetailsResponse } from "~/services/auth-service/auth-service";
import * as jsSHA from "jssha";

const getRemoteHostBaseUrl = async (remoteHost: string): Promise<string | null> => {
    let url = remoteHost;
    if (url.indexOf('://') < 0) {
        url = 'https://' + url;
    }
    const origin = new URL(url).origin;
    let baseUrl: string | null = null;

    try {
        const resp = await Axios.get<Config>(`${origin}/${DISCOVERY_URL}`);
        baseUrl = resp.data.baseUrl;
    } catch (err) {
        try {
            const resp = await Axios.get<any>(`${origin}/status.json`);
            baseUrl = resp.data.apiBaseURL;
        } catch (err) {
        }
    }

    if (baseUrl && baseUrl[baseUrl.length - 1] === '/') {
        baseUrl = baseUrl.substr(0, baseUrl.length - 1);
    }

    return baseUrl;
};

const getUserDetails = async (baseUrl: string, token: string): Promise<UserDetailsResponse> => {
    const resp = await Axios.get<UserDetailsResponse>(`${baseUrl}/users/current`, {
        headers: {
            Authorization: `OAuth2 ${token}`
        }
    });
    return resp.data;
};

const getTokenUuid = async (baseUrl: string, token: string): Promise<string> => {
    if (token.startsWith("v2/")) {
        const uuid = token.split("/")[1];
        return Promise.resolve(uuid);
    }

    const resp = await Axios.get(`${baseUrl}api_client_authorizations`, {
        headers: {
            Authorization: `OAuth2 ${token}`
        },
        data: {
            filters: JSON.stringify([['api_token', '=', token]])
        }
    });

    return resp.data.items[0].uuid;
};

const getSaltedToken = (clusterId: string, tokenUuid: string, token: string) => {
    const shaObj = new jsSHA("SHA-1", "TEXT");
    let secret = token;
    if (token.startsWith("v2/")) {
        secret = token.split("/")[2];
    }
    shaObj.setHMACKey(secret, "TEXT");
    shaObj.update(clusterId);
    const hmac = shaObj.getHMAC("HEX");
    return `v2/${tokenUuid}/${hmac}`;
};

const clusterLogin = async (clusterId: string, baseUrl: string, activeSession: Session): Promise<{ user: User, token: string }> => {
    const tokenUuid = await getTokenUuid(activeSession.baseUrl, activeSession.token);
    const saltedToken = getSaltedToken(clusterId, tokenUuid, activeSession.token);
    const user = await getUserDetails(baseUrl, saltedToken);
    return {
        user: {
            firstName: user.first_name,
            lastName: user.last_name,
            uuid: user.uuid,
            ownerUuid: user.owner_uuid,
            email: user.email,
            isAdmin: user.is_admin,
            identityUrl: user.identity_url,
            prefs: user.prefs
        },
        token: saltedToken
    };
};

export const getActiveSession = (sessions: Session[]): Session | undefined => sessions.find(s => s.active);

export const validateCluster = async (remoteHost: string, clusterId: string, activeSession: Session): Promise<{ user: User; token: string, baseUrl: string }> => {
    const baseUrl = await getRemoteHostBaseUrl(remoteHost);
    if (!baseUrl) {
        return Promise.reject(`Could not find base url for ${remoteHost}`);
    }
    const { user, token } = await clusterLogin(clusterId, baseUrl, activeSession);
    return { baseUrl, user, token };
};

export const validateSession = (session: Session, activeSession: Session) =>
    async (dispatch: Dispatch): Promise<Session> => {
        dispatch(authActions.UPDATE_SESSION({ ...session, status: SessionStatus.BEING_VALIDATED }));
        session.loggedIn = false;
        try {
            const { baseUrl, user, token } = await validateCluster(session.remoteHost, session.clusterId, activeSession);
            session.baseUrl = baseUrl;
            session.token = token;
            session.email = user.email;
            session.username = getUserFullname(user);
            session.loggedIn = true;
        } catch {
            session.loggedIn = false;
        } finally {
            session.status = SessionStatus.VALIDATED;
            dispatch(authActions.UPDATE_SESSION(session));
        }
        return session;
    };

export const validateSessions = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const sessions = getState().auth.sessions;
        const activeSession = getActiveSession(sessions);
        if (activeSession) {
            dispatch(progressIndicatorActions.START_WORKING("sessionsValidation"));
            for (const session of sessions) {
                if (session.status === SessionStatus.INVALIDATED) {
                    await dispatch(validateSession(session, activeSession));
                }
            }
            services.authService.saveSessions(sessions);
            dispatch(progressIndicatorActions.STOP_WORKING("sessionsValidation"));
        }
    };

export const addSession = (remoteHost: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const sessions = getState().auth.sessions;
        const activeSession = getActiveSession(sessions);
        if (activeSession) {
            const clusterId = remoteHost.match(/^(\w+)\./)![1];
            if (sessions.find(s => s.clusterId === clusterId)) {
                return Promise.reject("Cluster already exists");
            }
            try {
                const { baseUrl, user, token } = await validateCluster(remoteHost, clusterId, activeSession);
                const session = {
                    loggedIn: true,
                    status: SessionStatus.VALIDATED,
                    active: false,
                    email: user.email,
                    username: getUserFullname(user),
                    remoteHost,
                    baseUrl,
                    clusterId,
                    token
                };

                dispatch(authActions.ADD_SESSION(session));
                services.authService.saveSessions(getState().auth.sessions);

                return session;
            } catch (e) {
            }
        }
        return Promise.reject("Could not validate cluster");
    };

export const toggleSession = (session: Session) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let s = { ...session };

        if (session.loggedIn) {
            s.loggedIn = false;
        } else {
            const sessions = getState().auth.sessions;
            const activeSession = getActiveSession(sessions);
            if (activeSession) {
                s = await dispatch<any>(validateSession(s, activeSession)) as Session;
            }
        }

        dispatch(authActions.UPDATE_SESSION(s));
        services.authService.saveSessions(getState().auth.sessions);
    };

export const initSessions = (authService: AuthService, config: Config, user: User) =>
    (dispatch: Dispatch<any>) => {
        const sessions = authService.buildSessions(config, user);
        authService.saveSessions(sessions);
        dispatch(authActions.SET_SESSIONS(sessions));
    };

export const loadSiteManagerPanel = () =>
    async (dispatch: Dispatch<any>) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'Site Manager' }]));
            dispatch(validateSessions());
        } catch (e) {
            return;
        }
    };
