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
import { Config, ClusterConfigJSON, CLUSTER_CONFIG_PATH, DISCOVERY_DOC_PATH, ARVADOS_API_PATH } from "~/common/config";
import { normalizeURLPath } from "~/common/url";
import { Session, SessionStatus } from "~/models/session";
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { AuthService, UserDetailsResponse } from "~/services/auth-service/auth-service";
import * as jsSHA from "jssha";

const getClusterInfo = async (origin: string): Promise<{ clusterId: string, baseURL: string } | null> => {
    // Try the new public config endpoint
    try {
        const config = (await Axios.get<ClusterConfigJSON>(`${origin}/${CLUSTER_CONFIG_PATH}`)).data;
        return {
            clusterId: config.ClusterID,
            baseURL: normalizeURLPath(`${config.Services.Controller.ExternalURL}/${ARVADOS_API_PATH}`)
        };
    } catch { }

    // Fall back to discovery document
    try {
        const config = (await Axios.get<any>(`${origin}/${DISCOVERY_DOC_PATH}`)).data;
        return {
            clusterId: config.uuidPrefix,
            baseURL: normalizeURLPath(config.baseUrl)
        };
    } catch { }

    return null;
};

const getRemoteHostInfo = async (remoteHost: string): Promise<{ clusterId: string, baseURL: string } | null> => {
    let url = remoteHost;
    if (url.indexOf('://') < 0) {
        url = 'https://' + url;
    }
    const origin = new URL(url).origin;

    // Maybe it is an API server URL, try fetching config and discovery doc
    let r = getClusterInfo(origin);
    if (r !== null) {
        return r;
    }

    // Maybe it is a Workbench2 URL, try getting config.json
    try {
        r = getClusterInfo((await Axios.get<any>(`${origin}/config.json`)).data.API_HOST);
        if (r !== null) {
            return r;
        }
    } catch { }

    // Maybe it is a Workbench1 URL, try getting status.json
    try {
        r = getClusterInfo((await Axios.get<any>(`${origin}/status.json`)).data.apiBaseURL);
        if (r !== null) {
            return r;
        }
    } catch { }

    return null;
};

const getUserDetails = async (baseUrl: string, token: string): Promise<UserDetailsResponse> => {
    const resp = await Axios.get<UserDetailsResponse>(`${baseUrl}/users/current`, {
        headers: {
            Authorization: `OAuth2 ${token}`
        }
    });
    return resp.data;
};

export const getSaltedToken = (clusterId: string, token: string) => {
    const shaObj = new jsSHA("SHA-1", "TEXT");
    const [ver, uuid, secret] = token.split("/");
    if (ver !== "v2") {
        throw new Error("Must be a v2 token");
    }
    let salted = secret;
    if (uuid.substr(0, 5) !== clusterId) {
        shaObj.setHMACKey(secret, "TEXT");
        shaObj.update(clusterId);
        salted = shaObj.getHMAC("HEX");
    }
    return `v2/${uuid}/${salted}`;
};

export const getActiveSession = (sessions: Session[]): Session | undefined => sessions.find(s => s.active);

export const validateCluster = async (remoteHost: string, useToken: string):
    Promise<{ user: User; token: string, baseUrl: string, clusterId: string }> => {

    const info = await getRemoteHostInfo(remoteHost);
    if (!info) {
        return Promise.reject(`Could not get config for ${remoteHost}`);
    }
    const saltedToken = getSaltedToken(info.clusterId, useToken);
    const user = await getUserDetails(info.baseURL, saltedToken);
    return {
        baseUrl: info.baseURL,
        user: {
            firstName: user.first_name,
            lastName: user.last_name,
            uuid: user.uuid,
            ownerUuid: user.owner_uuid,
            email: user.email,
            isAdmin: user.is_admin,
            isActive: user.is_active,
            username: user.username,
            prefs: user.prefs
        },
        token: saltedToken,
        clusterId: info.clusterId
    };
};

export const validateSession = (session: Session, activeSession: Session) =>
    async (dispatch: Dispatch): Promise<Session> => {
        dispatch(authActions.UPDATE_SESSION({ ...session, status: SessionStatus.BEING_VALIDATED }));
        session.loggedIn = false;

        const setupSession = (baseUrl: string, user: User, token: string) => {
            session.baseUrl = baseUrl;
            session.token = token;
            session.email = user.email;
            session.uuid = user.uuid;
            session.name = getUserFullname(user);
            session.loggedIn = true;
        };

        try {
            const { baseUrl, user, token } = await validateCluster(session.remoteHost, session.token);
            setupSession(baseUrl, user, token);
        } catch {
            try {
                const { baseUrl, user, token } = await validateCluster(session.remoteHost, activeSession.token);
                setupSession(baseUrl, user, token);
            } catch { }
        }

        session.status = SessionStatus.VALIDATED;
        dispatch(authActions.UPDATE_SESSION(session));

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

export const addSession = (remoteHost: string, token?: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const sessions = getState().auth.sessions;
        const activeSession = getActiveSession(sessions);
        let useToken: string | null = null;
        if (token) {
            useToken = token;
        } else if (activeSession) {
            useToken = activeSession.token;
        }

        if (useToken) {
            try {
                const { baseUrl, user, token, clusterId } = await validateCluster(remoteHost, useToken);
                const session = {
                    loggedIn: true,
                    status: SessionStatus.VALIDATED,
                    active: false,
                    email: user.email,
                    name: getUserFullname(user),
                    uuid: user.uuid,
                    remoteHost,
                    baseUrl,
                    clusterId,
                    token
                };

                if (sessions.find(s => s.clusterId === clusterId)) {
                    dispatch(authActions.UPDATE_SESSION(session));
                } else {
                    dispatch(authActions.ADD_SESSION(session));
                }
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
        dispatch(validateSessions());
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
