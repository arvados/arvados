// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getUserFullname, User, UserPrefs } from '~/models/user';
import { AxiosInstance } from "axios";
import { ApiActions } from "~/services/api/api-actions";
import * as uuid from "uuid/v4";
import { Session, SessionStatus } from "~/models/session";
import { Config } from "~/common/config";
import { uniqBy } from "lodash";

export const API_TOKEN_KEY = 'apiToken';
export const USER_EMAIL_KEY = 'userEmail';
export const USER_FIRST_NAME_KEY = 'userFirstName';
export const USER_LAST_NAME_KEY = 'userLastName';
export const USER_UUID_KEY = 'userUuid';
export const USER_OWNER_UUID_KEY = 'userOwnerUuid';
export const USER_IS_ADMIN = 'isAdmin';
export const USER_IS_ACTIVE = 'isActive';
export const USER_USERNAME = 'username';
export const USER_PREFS = 'prefs';
export const HOME_CLUSTER = 'homeCluster';

export interface UserDetailsResponse {
    email: string;
    first_name: string;
    last_name: string;
    uuid: string;
    owner_uuid: string;
    is_admin: boolean;
    is_active: boolean;
    username: string;
    prefs: UserPrefs;
}

export class AuthService {

    constructor(
        protected apiClient: AxiosInstance,
        protected baseUrl: string,
        protected actions: ApiActions) { }

    public saveApiToken(token: string) {
        localStorage.setItem(API_TOKEN_KEY, token);
        localStorage.setItem(HOME_CLUSTER, token.split('/')[1].substr(0, 5));
    }

    public removeApiToken() {
        localStorage.removeItem(API_TOKEN_KEY);
    }

    public getApiToken() {
        return localStorage.getItem(API_TOKEN_KEY) || undefined;
    }

    public getHomeCluster() {
        return localStorage.getItem(HOME_CLUSTER) || undefined;
    }

    public removeUser() {
        localStorage.removeItem(USER_EMAIL_KEY);
        localStorage.removeItem(USER_FIRST_NAME_KEY);
        localStorage.removeItem(USER_LAST_NAME_KEY);
        localStorage.removeItem(USER_UUID_KEY);
        localStorage.removeItem(USER_OWNER_UUID_KEY);
        localStorage.removeItem(USER_IS_ADMIN);
        localStorage.removeItem(USER_IS_ACTIVE);
        localStorage.removeItem(USER_USERNAME);
        localStorage.removeItem(USER_PREFS);
    }

    public login(uuidPrefix: string, homeCluster: string, loginCluster: string, remoteHosts: { [key: string]: string }) {
        const currentUrl = `${window.location.protocol}//${window.location.host}/token`;
        const homeClusterHost = remoteHosts[homeCluster];
        window.location.assign(`https://${homeClusterHost}/login?${(uuidPrefix !== homeCluster && homeCluster !== loginCluster) ? "remote=" + uuidPrefix + "&" : ""}return_to=${currentUrl}`);
    }

    public logout() {
        const currentUrl = `${window.location.protocol}//${window.location.host}`;
        window.location.assign(`${this.baseUrl || ""}/logout?return_to=${currentUrl}`);
    }

    public getUserDetails = (): Promise<User> => {
        const reqId = uuid();
        this.actions.progressFn(reqId, true);
        return this.apiClient
            .get<UserDetailsResponse>('/users/current')
            .then(resp => {
                this.actions.progressFn(reqId, false);
                const prefs = resp.data.prefs.profile ? resp.data.prefs : { profile: {} };
                return {
                    email: resp.data.email,
                    firstName: resp.data.first_name,
                    lastName: resp.data.last_name,
                    uuid: resp.data.uuid,
                    ownerUuid: resp.data.owner_uuid,
                    isAdmin: resp.data.is_admin,
                    isActive: resp.data.is_active,
                    username: resp.data.username,
                    prefs
                };
            })
            .catch(e => {
                this.actions.progressFn(reqId, false);
                this.actions.errorFn(reqId, e);
                throw e;
            });
    }

    public getSessions(): Session[] {
        try {
            const sessions = JSON.parse(localStorage.getItem("sessions") || '');
            return sessions;
        } catch {
            return [];
        }
    }

    public saveSessions(sessions: Session[]) {
        localStorage.setItem("sessions", JSON.stringify(sessions));
    }

    public buildSessions(cfg: Config, user?: User) {
        const currentSession = {
            clusterId: cfg.uuidPrefix,
            remoteHost: cfg.rootUrl,
            baseUrl: cfg.baseUrl,
            name: getUserFullname(user),
            email: user ? user.email : '',
            token: this.getApiToken(),
            loggedIn: true,
            active: true,
            uuid: user ? user.uuid : '',
            status: SessionStatus.VALIDATED
        } as Session;
        const localSessions = this.getSessions().map(s => ({
            ...s,
            active: false,
            status: SessionStatus.INVALIDATED
        }));

        const cfgSessions = Object.keys(cfg.remoteHosts).map(clusterId => {
            const remoteHost = cfg.remoteHosts[clusterId];
            return {
                clusterId,
                remoteHost,
                baseUrl: '',
                name: '',
                email: '',
                token: '',
                loggedIn: false,
                active: false,
                uuid: '',
                status: SessionStatus.INVALIDATED
            } as Session;
        });
        const sessions = [currentSession]
            .concat(cfgSessions)
            .concat(localSessions)
            .filter((r: Session) => r.clusterId !== "*");

        const uniqSessions = uniqBy(sessions, 'clusterId');

        return uniqSessions;
    }
}
