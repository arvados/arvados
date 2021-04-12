// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { getNewExtraToken } from "../auth/auth-action";

export const COLLECTION_WEBDAV_S3_DIALOG_NAME = 'collectionWebdavS3Dialog';

export interface WebDavS3InfoDialogData {
    uuid: string;
    token: string;
    downloadUrl: string;
    collectionsUrl: string;
    localCluster: string;
    username: string;
    activeTab: number;
    collectionName?: string;
    setActiveTab: (event: any, tabNr: number) => void;
}

export const openWebDavS3InfoDialog = (uuid: string, activeTab?: number) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        await dispatch<any>(getNewExtraToken(true));
        dispatch(dialogActions.OPEN_DIALOG({
            id: COLLECTION_WEBDAV_S3_DIALOG_NAME,
            data: {
                title: 'Access Collection using WebDAV or S3',
                token: getState().auth.extraApiToken || getState().auth.apiToken,
                downloadUrl: getState().auth.config.keepWebServiceUrl,
                collectionsUrl: getState().auth.config.keepWebInlineServiceUrl,
                localCluster: getState().auth.localCluster,
                username: getState().auth.user!.username,
                activeTab: activeTab || 0,
                collectionName: (getState().collectionPanel.item || {} as any).name,
                setActiveTab: (event: any, tabNr: number) => dispatch<any>(openWebDavS3InfoDialog(uuid, tabNr)),
                uuid
            }
        }));
    };
