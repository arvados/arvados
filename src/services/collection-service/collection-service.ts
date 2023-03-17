// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionResource, defaultCollectionSelectedFields } from "models/collection";
import { AxiosInstance } from "axios";
import { CollectionFile, CollectionDirectory } from "models/collection-file";
import { WebDAV } from "common/webdav";
import { AuthService } from "../auth-service/auth-service";
import { extractFilesData } from "./collection-service-files-response";
import { TrashableResourceService } from "services/common-service/trashable-resource-service";
import { ApiActions } from "services/api/api-actions";
import { Session } from "models/session";
import { CommonService } from "services/common-service/common-service";

export type UploadProgress = (fileId: number, loaded: number, total: number, currentTime: number) => void;

export const emptyCollectionPdh = 'd41d8cd98f00b204e9800998ecf8427e+0';

export class CollectionService extends TrashableResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private webdavClient: WebDAV, private authService: AuthService, actions: ApiActions) {
        super(serverApi, "collections", actions, [
            'fileCount',
            'fileSizeTotal',
            'replicationConfirmed',
            'replicationConfirmedAt',
            'storageClassesConfirmed',
            'storageClassesConfirmedAt',
            'unsignedManifestText',
            'version',
        ]);
    }

    async get(uuid: string, showErrors?: boolean, select?: string[], session?: Session) {
        super.validateUuid(uuid);
        const selectParam = select || defaultCollectionSelectedFields;
        return super.get(uuid, showErrors, selectParam, session);
    }

    create(data?: Partial<CollectionResource>, showErrors?: boolean) {
        return super.create({ ...data, preserveVersion: true }, showErrors);
    }

    update(uuid: string, data: Partial<CollectionResource>, showErrors?: boolean) {
        const select = [...Object.keys(data), 'version', 'modifiedAt'];
        return super.update(uuid, { ...data, preserveVersion: true }, showErrors, select);
    }

    async files(uuid: string) {
        const request = await this.webdavClient.propfind(`c=${uuid}`);
        if (request.responseXML != null) {
            return extractFilesData(request.responseXML);
        }
        return Promise.reject();
    }

    private combineFilePath(parts: string[]) {
        return parts.reduce((path, part) => {
            // Trim leading and trailing slashes
            const trimmedPart = part.split('/').filter(Boolean).join('/');
            if (trimmedPart.length) {
                const separator = path.endsWith('/') ? '' : '/';
                return `${path}${separator}${trimmedPart}`;
            } else {
                return path;
            }
        }, "/");
    }

    private replaceFiles(collectionUuid: string, fileMap: {}, showErrors?: boolean) {
        const payload = {
            collection: {
                preserve_version: true
            },
            replace_files: fileMap
        };

        return CommonService.defaultResponse(
            this.serverApi
                .put<CollectionResource>(`/${this.resourceType}/${collectionUuid}`, payload),
            this.actions,
            true, // mapKeys
            showErrors
        );
    }

    async uploadFiles(collectionUuid: string, files: File[], onProgress?: UploadProgress, targetLocation: string = '') {
        if (collectionUuid === "" || files.length === 0) { return; }
        // files have to be uploaded sequentially
        for (let idx = 0; idx < files.length; idx++) {
            await this.uploadFile(collectionUuid, files[idx], idx, onProgress, targetLocation);
        }
        await this.update(collectionUuid, { preserveVersion: true });
    }

    async renameFile(collectionUuid: string, collectionPdh: string, oldPath: string, newPath: string) {
        return this.replaceFiles(collectionUuid, {
            [this.combineFilePath([newPath])]: `${collectionPdh}${this.combineFilePath([oldPath])}`,
            [this.combineFilePath([oldPath])]: '',
        });
    }

    extendFileURL = (file: CollectionDirectory | CollectionFile) => {
        const baseUrl = this.webdavClient.getBaseUrl().endsWith('/')
            ? this.webdavClient.getBaseUrl().slice(0, -1)
            : this.webdavClient.getBaseUrl();
        const apiToken = this.authService.getApiToken();
        const encodedApiToken = apiToken ? encodeURI(apiToken) : '';
        const userApiToken = `/t=${encodedApiToken}/`;
        const splittedPrevFileUrl = file.url.split('/');
        const url = `${baseUrl}/${splittedPrevFileUrl[1]}${userApiToken}${splittedPrevFileUrl.slice(2).join('/')}`;
        return {
            ...file,
            url
        };
    }

    async getFileContents(file: CollectionFile) {
        return (await this.webdavClient.get(`c=${file.id}`)).response;
    }

    private async uploadFile(collectionUuid: string, file: File, fileId: number, onProgress: UploadProgress = () => { return; }, targetLocation: string = '') {
        const fileURL = `c=${targetLocation !== '' ? targetLocation : collectionUuid}/${file.name}`.replace('//', '/');
        const requestConfig = {
            headers: {
                'Content-Type': 'text/octet-stream'
            },
            onUploadProgress: (e: ProgressEvent) => {
                onProgress(fileId, e.loaded, e.total, Date.now());
            },
        };
        return this.webdavClient.upload(fileURL, [file], requestConfig);
    }

    deleteFiles(collectionUuid: string, files: string[], showErrors?: boolean) {
        const optimizedFiles = files
            .sort((a, b) => a.length - b.length)
            .reduce((acc, currentPath) => {
                const parentPathFound = acc.find((parentPath) => currentPath.indexOf(`${parentPath}/`) > -1);

                if (!parentPathFound) {
                    return [...acc, currentPath];
                }

                return acc;
            }, []);

        const fileMap = optimizedFiles.reduce((obj, filePath) => {
            return {
                ...obj,
                [this.combineFilePath([filePath])]: ''
            }
        }, {})

        return this.replaceFiles(collectionUuid, fileMap, showErrors);
    }

    copyFiles(sourcePdh: string, files: string[], destinationCollectionUuid: string, destinationPath: string, showErrors?: boolean) {
        const fileMap = files.reduce((obj, sourceFile) => {
            const sourceFileName = sourceFile.split('/').filter(Boolean).slice(-1).join("");
            return {
                ...obj,
                [this.combineFilePath([destinationPath, sourceFileName])]: `${sourcePdh}${this.combineFilePath([sourceFile])}`
            };
        }, {});

        return this.replaceFiles(destinationCollectionUuid, fileMap, showErrors);
    }

    moveFiles(sourceUuid: string, sourcePdh: string, files: string[], destinationCollectionUuid: string, destinationPath: string, showErrors?: boolean) {
        if (sourceUuid === destinationCollectionUuid) {
            const fileMap = files.reduce((obj, sourceFile) => {
                const sourceFileName = sourceFile.split('/').filter(Boolean).slice(-1).join("");
                return {
                    ...obj,
                    [this.combineFilePath([destinationPath, sourceFileName])]: `${sourcePdh}${this.combineFilePath([sourceFile])}`,
                    [this.combineFilePath([sourceFile])]: '',
                };
            }, {});

            return this.replaceFiles(sourceUuid, fileMap, showErrors)
        } else {
            return this.copyFiles(sourcePdh, files, destinationCollectionUuid, destinationPath, showErrors)
                .then(() => {
                    return this.deleteFiles(sourceUuid, files, showErrors);
                });
        }
    }

    createDirectory(collectionUuid: string, path: string, showErrors?: boolean) {
        const fileMap = {[this.combineFilePath([path])]: emptyCollectionPdh};

        return this.replaceFiles(collectionUuid, fileMap, showErrors);
    }

}
