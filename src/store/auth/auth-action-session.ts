// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { setBreadcrumbs } from "store/breadcrumbs/breadcrumbs-actions";
import { RootState } from "store/store";
import { ServiceRepository, createServices, setAuthorizationHeader } from "services/services";
import Axios, { AxiosInstance } from "axios";
import { User, getUserDisplayName } from "models/user";
import { authActions } from "store/auth/auth-action";
import {
    Config, ClusterConfigJSON, CLUSTER_CONFIG_PATH, DISCOVERY_DOC_PATH,
    buildConfig, mockClusterConfigJSON
} from "common/config";
import { normalizeURLPath } from "common/url";
import { Session, SessionStatus } from "models/session";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { AuthService } from "services/auth-service/auth-service";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import jsSHA from "jssha";

const getClusterConfig = async (origin: string, apiClient: AxiosInstance): Promise<Config | null> => {
    let configFromDD: Config | undefined;
    try {
        const dd = (await apiClient.get<any>(`${origin}/${DISCOVERY_DOC_PATH}`)).data;
        configFromDD = {
            baseUrl: normalizeURLPath(dd.baseUrl),
            keepWebServiceUrl: dd.keepWebServiceUrl,
            keepWebInlineServiceUrl: dd.keepWebInlineServiceUrl,
            remoteHosts: dd.remoteHosts,
            rootUrl: dd.rootUrl,
            uuidPrefix: dd.uuidPrefix,
            websocketUrl: dd.websocketUrl,
            workbenchUrl: dd.workbenchUrl,
            workbench2Url: dd.workbench2Url,
            loginCluster: "",
            vocabularyUrl: "",
            fileViewersConfigUrl: "",
            clusterConfig: mockClusterConfigJSON({}),
            apiRevision: parseInt(dd.revision, 10),
        };
    } catch { }

    // Try public config endpoint
    try {
        const config = (await apiClient.get<ClusterConfigJSON>(`${origin}/${CLUSTER_CONFIG_PATH}`)).data;
        return { ...buildConfig(config), apiRevision: configFromDD ? configFromDD.apiRevision : 0 };
    } catch { }

    // Fall back to discovery document
    if (configFromDD !== undefined) {
        return configFromDD;
    }

    return null;
};

export const getRemoteHostConfig = async (remoteHost: string, useApiClient?: AxiosInstance): Promise<Config | null> => {
    const apiClient = useApiClient || Axios.create({ headers: {} });

    let url = remoteHost;
    if (url.indexOf('://') < 0) {
        url = 'https://' + url;
    }
    const origin = new URL(url).origin;

    // Maybe it is an API server URL, try fetching config and discovery doc
    let r = await getClusterConfig(origin, apiClient);
    if (r !== null) {
        return r;
    }

    // Maybe it is a Workbench2 URL, try getting config.json
    try {
        r = await getClusterConfig((await apiClient.get<any>(`${origin}/config.json`)).data.API_HOST, apiClient);
        if (r !== null) {
            return r;
        }
    } catch { }

    // Maybe it is a Workbench1 URL, try getting status.json
    try {
        r = await getClusterConfig((await apiClient.get<any>(`${origin}/status.json`)).data.apiBaseURL, apiClient);
        if (r !== null) {
            return r;
        }
    } catch { }

    return null;
};

const invalidV2Token = "Must be a v2 token";

export const getSaltedToken = (clusterId: string, token: string) => {
    const shaObj = new jsSHA("SHA-1", "TEXT");
    const [ver, uuid, secret] = token.split("/");
    if (ver !== "v2") {
        throw new Error(invalidV2Token);
    }
    let salted = secret;
    if (uuid.substring(0, 5) !== clusterId) {
        shaObj.setHMACKey(secret, "TEXT");
        shaObj.update(clusterId);
        salted = shaObj.getHMAC("HEX");
    }
    return `v2/${uuid}/${salted}`;
};

export const getActiveSession = (sessions: Session[]): Session | undefined => sessions.find(s => s.active);

export const validateCluster = async (config: Config, useToken: string):
    Promise<{ user: User; token: string }> => {

    const saltedToken = getSaltedToken(config.uuidPrefix, useToken);

    const svc = createServices(config, { progressFn: () => { }, errorFn: () => { } });
    setAuthorizationHeader(svc, saltedToken);

    const user = await svc.authService.getUserDetails(false);
    return {
        user,
        token: saltedToken,
    };
};

export const validateSession = (session: Session, activeSession: Session, useApiClient?: AxiosInstance) =>
    async (dispatch: Dispatch): Promise<Session> => {
        dispatch(authActions.UPDATE_SESSION({ ...session, status: SessionStatus.BEING_VALIDATED }));
        session.loggedIn = false;

        const setupSession = (baseUrl: string, user: User, token: string, apiRevision: number) => {
            session.baseUrl = baseUrl;
            session.token = token;
            session.email = user.email;
            session.userIsActive = user.isActive;
            session.uuid = user.uuid;
            session.name = getUserDisplayName(user);
            session.loggedIn = true;
            session.apiRevision = apiRevision;
        };

        let fail: Error | null = null;
        const config = await getRemoteHostConfig(session.remoteHost, useApiClient);
        if (config !== null) {
            dispatch(authActions.REMOTE_CLUSTER_CONFIG({ config }));
            try {
                const { user, token } = await validateCluster(config, session.token);
                setupSession(config.baseUrl, user, token, config.apiRevision);
            } catch (e) {
                fail = new Error(`Getting current user for ${session.remoteHost}: ${e.message}`);
                try {
                    const { user, token } = await validateCluster(config, activeSession.token);
                    setupSession(config.baseUrl, user, token, config.apiRevision);
                    fail = null;
                } catch (e2) {
                    if (e.message === invalidV2Token) {
                        fail = new Error(`Getting current user for ${session.remoteHost}: ${e2.message}`);
                    }
                }
            }
        } else {
            fail = new Error(`Could not get config for ${session.remoteHost}`);
        }
        session.status = SessionStatus.VALIDATED;
        dispatch(authActions.UPDATE_SESSION(session));

        if (fail) {
            throw fail;
        }

        return session;
    };

export const validateSessions = (useApiClient?: AxiosInstance) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const sessions = getState().auth.sessions;
        const activeSession = getActiveSession(sessions);
        if (activeSession) {
            dispatch(progressIndicatorActions.START_WORKING("sessionsValidation"));
            for (const session of sessions) {
                if (session.status === SessionStatus.INVALIDATED) {
                    try {
			/* Here we are dispatching a function, not an
			   action.  This is legal (it calls the
			   function with a 'Dispatch' object as the
			   first parameter) but the typescript
			   annotations don't understand this case, so
			   we get an error from typescript unless
			   override it using Dispatch<any>.  This
			   pattern is used in a bunch of different
			   places in Workbench2. */
                        await dispatch(validateSession(session, activeSession, useApiClient));
                    } catch (e) {
                        // Don't do anything here.  User may get
                        // spammed with multiple messages that are not
                        // helpful.  They can see the individual
                        // errors by going to site manager and trying
                        // to toggle the session.
                    }
                }
            }
            services.authService.saveSessions(getState().auth.sessions);
            dispatch(progressIndicatorActions.STOP_WORKING("sessionsValidation"));
        }
    };

export const addRemoteConfig = (remoteHost: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const config = await getRemoteHostConfig(remoteHost);
        if (!config) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Could not get config for ${remoteHost}`,
                kind: SnackbarKind.ERROR
            }));
            return;
        }
        dispatch(authActions.REMOTE_CLUSTER_CONFIG({ config }));
    };

export const addSession = (remoteHost: string, token?: string, sendToLogin?: boolean) =>
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
            const config = await getRemoteHostConfig(remoteHost);
            if (!config) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: `Could not get config for ${remoteHost}`,
                    kind: SnackbarKind.ERROR
                }));
                return;
            }

            try {
                dispatch(authActions.REMOTE_CLUSTER_CONFIG({ config }));
                const { user, token } = await validateCluster(config, useToken);
                const session = {
                    loggedIn: true,
                    status: SessionStatus.VALIDATED,
                    active: false,
                    email: user.email,
                    userIsActive: user.isActive,
                    name: getUserDisplayName(user),
                    uuid: user.uuid,
                    baseUrl: config.baseUrl,
                    clusterId: config.uuidPrefix,
                    remoteHost,
                    token,
                    apiRevision: config.apiRevision,
                };

                if (sessions.find(s => s.clusterId === config.uuidPrefix)) {
                    await dispatch(authActions.UPDATE_SESSION(session));
                } else {
                    await dispatch(authActions.ADD_SESSION(session));
                }
                services.authService.saveSessions(getState().auth.sessions);

                return session;
            } catch {
                if (sendToLogin) {
                    const rootUrl = new URL(config.baseUrl);
                    rootUrl.pathname = "";
                    window.location.href = `${rootUrl.toString()}/login?return_to=` + encodeURI(`${window.location.protocol}//${window.location.host}/add-session?baseURL=` + encodeURI(rootUrl.toString()));
                    return;
                }
            }
        }
        return Promise.reject(new Error("Could not validate cluster"));
    };


export const removeSession = (clusterId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        await dispatch(authActions.REMOVE_SESSION(clusterId));
        services.authService.saveSessions(getState().auth.sessions);
    };

export const toggleSession = (session: Session) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const s: Session = { ...session };

        if (session.loggedIn) {
            s.loggedIn = false;
            dispatch(authActions.UPDATE_SESSION(s));
        } else {
            const sessions = getState().auth.sessions;
            const activeSession = getActiveSession(sessions);
            if (activeSession) {
                try {
                    await dispatch(validateSession(s, activeSession));
                } catch (e) {
                    dispatch(snackbarActions.OPEN_SNACKBAR({
                        message: e.message,
                        kind: SnackbarKind.ERROR
                    }));
                    s.loggedIn = false;
                    dispatch(authActions.UPDATE_SESSION(s));
                }
            }
        }

        services.authService.saveSessions(getState().auth.sessions);
    };

export const initSessions = (authService: AuthService, config: Config, user: User) =>
    (dispatch: Dispatch<any>) => {
        const sessions = authService.buildSessions(config, user);
        dispatch(authActions.SET_SESSIONS(sessions));
        dispatch(validateSessions(authService.getApiClient()));
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
