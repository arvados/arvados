// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionResource } from "~/models/collection";
import { AxiosInstance } from "axios";
import { CollectionFile, CollectionDirectory } from "~/models/collection-file";
import { WebDAV } from "~/common/webdav";
import { AuthService } from "../auth-service/auth-service";
import { mapTreeValues } from "~/models/tree";
import { parseFilesResponse } from "./collection-service-files-response";
import { TrashableResourceService } from "~/services/common-service/trashable-resource-service";
import { ApiActions } from "~/services/api/api-actions";

export type UploadProgress = (fileId: number, loaded: number, total: number, currentTime: number) => void;

export class CollectionService extends TrashableResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private webdavClient: WebDAV, private authService: AuthService, actions: ApiActions) {
        super(serverApi, "collections", actions);
    }

    async files(uuid: string) {
        const request = await this.webdavClient.propfind(`c=${uuid}`);
        if (request.responseXML != null) {
            const filesTree = parseFilesResponse(request.responseXML);
            return mapTreeValues(this.extendFileURL)(filesTree);
        }
        return Promise.reject();
    }

    async deleteFiles(collectionUuid: string, filePaths: string[]) {
        for (const path of filePaths) {
            const splittedPath = path.split('/');
            if (collectionUuid) {
                await this.webdavClient.delete(`c=${collectionUuid}/${splittedPath[1]}`);
            } else {
                await this.webdavClient.delete(`c=${collectionUuid}${path}`);
            }
        }
    }

    async uploadFiles(collectionUuid: string, files: File[], onProgress?: UploadProgress) {
        // files have to be uploaded sequentially
        for (let idx = 0; idx < files.length; idx++) {
            await this.uploadFile(collectionUuid, files[idx], idx, onProgress);
        }
    }

    moveFile(collectionUuid: string, oldPath: string, newPath: string) {
        return this.webdavClient.move(
            `c=${collectionUuid}${oldPath}`,
            `c=${collectionUuid}${encodeURI(newPath)}`
        );
    }

    private extendFileURL = (file: CollectionDirectory | CollectionFile) => {
        const baseUrl = this.webdavClient.defaults.baseURL.endsWith('/')
            ? this.webdavClient.defaults.baseURL.slice(0, -1)
            : this.webdavClient.defaults.baseURL;
        const apiToken = this.authService.getApiToken();
        const splittedApiToken = apiToken ? apiToken.split('/') : [];
        const userApiToken = `/t=${splittedApiToken[2]}/`;
        const splittedPrevFileUrl = file.url.split('/');
        const url = `${baseUrl}/${splittedPrevFileUrl[1]}${userApiToken}${splittedPrevFileUrl[2]}`;
        return {
            ...file,
            url
        };
    }

    private async uploadFile(collectionUuid: string, file: File, fileId: number, onProgress: UploadProgress = () => { return; }) {
        const fileURL = `c=${collectionUuid}/${file.name}`;
        const requestConfig = {
            headers: {
                'Content-Type': 'text/octet-stream'
            },
            onUploadProgress: (e: ProgressEvent) => {
                onProgress(fileId, e.loaded, e.total, Date.now());
            }
        };
        return this.webdavClient.upload(fileURL, [file], requestConfig);
    }
}
