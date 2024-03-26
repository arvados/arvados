// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ofType, unionize } from 'common/unionize';
import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { dialogActions } from 'store/dialog/dialog-actions';
import { CollectionResource } from "models/collection";
import { SshKeyResource } from 'models/ssh-key';
import { User } from "models/user";
import { Session } from "models/session";
import { Config } from 'common/config';
import { createServices, setAuthorizationHeader } from "services/services";
import { getTokenV2 } from 'models/api-client-authorization';

export const COLLECTION_WEBDAV_S3_DIALOG_NAME = 'collectionWebdavS3Dialog';

export interface WebDavS3InfoDialogData {
    uuid: string;
    token: string;
    downloadUrl: string;
    collectionsUrl: string;
    localCluster: string;
    username: string;
    activeTab: number;
    collectionName: string;
    setActiveTab: (event: any, tabNr: number) => void;
}

export const openWebDavS3InfoDialog = (uuid: string, activeTab?: number) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        await dispatch<any>(getNewExtraToken(true));
        dispatch(dialogActions.OPEN_DIALOG({
            id: COLLECTION_WEBDAV_S3_DIALOG_NAME,
            data: {
                title: 'Open with 3rd party client',
                token: getState().auth.extraApiToken || getState().auth.apiToken,
                downloadUrl: getState().auth.config.keepWebServiceUrl,
                collectionsUrl: getState().auth.config.keepWebInlineServiceUrl,
                localCluster: getState().auth.localCluster,
                username: getState().auth.user!.username,
                activeTab: activeTab || 0,
                collectionName: (getState().resources[uuid] as CollectionResource).name,
                setActiveTab: (event: any, tabNr: number) => dispatch<any>(openWebDavS3InfoDialog(uuid, tabNr)),
                uuid
            }
        }));
    };

const authActions = unionize({
    LOGIN: {},
    LOGOUT: ofType<{ deleteLinkData: boolean, preservePath: boolean }>(),
    SET_CONFIG: ofType<{ config: Config }>(),
    SET_EXTRA_TOKEN: ofType<{ extraApiToken: string, extraApiTokenExpiration?: Date }>(),
    RESET_EXTRA_TOKEN: {},
    INIT_USER: ofType<{ user: User, token: string, tokenExpiration?: Date, tokenLocation?: string }>(),
    USER_DETAILS_REQUEST: {},
    USER_DETAILS_SUCCESS: ofType<User>(),
    SET_SSH_KEYS: ofType<SshKeyResource[]>(),
    ADD_SSH_KEY: ofType<SshKeyResource>(),
    REMOVE_SSH_KEY: ofType<string>(),
    SET_HOME_CLUSTER: ofType<string>(),
    SET_SESSIONS: ofType<Session[]>(),
    ADD_SESSION: ofType<Session>(),
    REMOVE_SESSION: ofType<string>(),
    UPDATE_SESSION: ofType<Session>(),
    REMOTE_CLUSTER_CONFIG: ofType<{ config: Config }>(),
});

const getConfig = (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Config => {
    const state = getState().auth;
    return state.remoteHostsConfig[state.localCluster];
};

const getNewExtraToken =
    (reuseStored: boolean = false) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const extraToken = getState().auth.extraApiToken;
        if (reuseStored && extraToken !== undefined) {
            const config = dispatch<any>(getConfig);
            const svc = createServices(config, { progressFn: () => {}, errorFn: () => {} });
            setAuthorizationHeader(svc, extraToken);
            try {
                // Check the extra token's validity before using it. Refresh its
                // expiration date just in case it changed.
                const client = await svc.apiClientAuthorizationService.get('current');
                dispatch(
                    authActions.SET_EXTRA_TOKEN({
                        extraApiToken: extraToken,
                        extraApiTokenExpiration: client.expiresAt ? new Date(client.expiresAt) : undefined,
                    })
                );
                return extraToken;
            } catch (e) {
                dispatch(authActions.RESET_EXTRA_TOKEN());
            }
        }
        const user = getState().auth.user;
        const loginCluster = getState().auth.config.clusterConfig.Login.LoginCluster;
        if (user === undefined) {
            return;
        }
        if (loginCluster !== '' && getState().auth.homeCluster !== loginCluster) {
            return;
        }
        try {
            // Do not show errors on the create call, cluster security configuration may not
            // allow token creation and there's no way to know that from workbench2 side in advance.
            const client = await services.apiClientAuthorizationService.create(undefined, false);
            const newExtraToken = getTokenV2(client);
            dispatch(
                authActions.SET_EXTRA_TOKEN({
                    extraApiToken: newExtraToken,
                    extraApiTokenExpiration: client.expiresAt ? new Date(client.expiresAt) : undefined,
                })
            );
            return newExtraToken;
        } catch {
            console.warn("Cannot create new tokens with the current token, probably because of cluster's security settings.");
            return;
        }
    };