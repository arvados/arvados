// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionResource, defaultCollectionSelectedFields } from "models/collection";
import { AxiosInstance, AxiosResponse } from "axios";
import { CollectionFile, CollectionDirectory } from "models/collection-file";
import { WebDAV } from "common/webdav";
import { AuthService } from "../auth-service/auth-service";
import { extractFilesData } from "./collection-service-files-response";
import { TrashableResourceService } from "services/common-service/trashable-resource-service";
import { ApiActions } from "services/api/api-actions";
import { Session } from "models/session";
import { CommonService } from "services/common-service/common-service";
import { snakeCase } from "lodash";
import { CommonResourceServiceError } from "services/common-service/common-resource-service";

export type UploadProgress = (fileId: number, loaded: number, total: number, currentTime: number) => void;
type CollectionPartialUpdateOrCreate =
    | (Partial<CollectionResource> & Pick<CollectionResource, "uuid">)
    | (Partial<CollectionResource> & Pick<CollectionResource, "ownerUuid">);

type ReplaceFilesPayload = {
    collection: Partial<CollectionResource>;
    replace_files: {[key: string]: string};
}

type FileWithRelativePath = File & { relativePath?: string, webkitRelativePath?: string };

export const emptyCollectionPdh = "d41d8cd98f00b204e9800998ecf8427e+0";
export const SOURCE_DESTINATION_EQUAL_ERROR_MESSAGE = "Source and destination cannot be the same";

export class CollectionService extends TrashableResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private keepWebdavClient: WebDAV, private authService: AuthService, actions: ApiActions) {
        super(serverApi, "collections", actions, [
            "fileCount",
            "fileSizeTotal",
            "replicationConfirmed",
            "replicationConfirmedAt",
            "storageClassesConfirmed",
            "storageClassesConfirmedAt",
            "unsignedManifestText",
            "version",
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
        const select = [...Object.keys(data), "version", "modifiedAt"];
        return super.update(uuid, { ...data, preserveVersion: true }, showErrors, select);
    }

    async files(uuid: string) {
        try {
            const request = await this.keepWebdavClient.propfind(`c=${uuid}`);
            if (request.responseXML != null) {
                return extractFilesData(request.responseXML);
            }
        } catch (e) {
            return Promise.reject(e);
        }
        return Promise.reject();
    }

    private combineFilePath(parts: string[]) {
        return parts.reduce((path, part) => {
            // Trim leading and trailing slashes
            const trimmedPart = part.split("/").filter(Boolean).join("/");
            if (trimmedPart.length) {
                const separator = path.endsWith("/") ? "" : "/";
                return `${path}${separator}${trimmedPart}`;
            } else {
                return path;
            }
        }, "/");
    }

    private replaceFiles(data: CollectionPartialUpdateOrCreate, fileMap: {}, showErrors?: boolean, preserveVersion: boolean = true) {
        const payload: ReplaceFilesPayload = {
            collection: {
                preserve_version: preserveVersion,
                ...CommonService.mapKeys(snakeCase)(data),
                // Don't send uuid in payload when creating
                uuid: undefined,
            },
            replace_files: fileMap,
        };
        if (data.uuid) {
            return CommonService.defaultResponse(
                this.serverApi.put<ReplaceFilesPayload, AxiosResponse<CollectionResource>>(`/${this.resourceType}/${data.uuid}`, payload),
                this.actions,
                true, // mapKeys
                showErrors
            );
        } else {
            return CommonService.defaultResponse(
                this.serverApi.post<ReplaceFilesPayload, AxiosResponse<CollectionResource>>(`/${this.resourceType}`, payload),
                this.actions,
                true, // mapKeys
                showErrors
            );
        }
    }

    async uploadFiles(collectionUuid: string, files: File[], onProgress?: UploadProgress, targetLocation: string = "") {
        if (collectionUuid === "" || files.length === 0) {
            return;
        }

        const isAnyNested = files.some(file => {
            const path = file[getPathKey(file)];
            return path && path.indexOf('/') > -1;
        });

        if (isAnyNested) {
            const existingDirPaths = new Set((await this.files(collectionUuid))
                .filter(f => f.type === 'directory')
                .map(f => f.id.split('/').slice(1).join('/')));

            const allTargetPaths = files.reduce((acc, file: FileWithRelativePath) => {
                const pathKey = getPathKey(file);
                if (file[pathKey] && file[pathKey]!.length > 0) {
                    acc.push(file[pathKey]!.split('/').slice(0, -1).join('/'));
                }
                return acc;
            }, [] as string[]).filter(path => path.length > 0);

            await this.createMinNecessaryDirs(collectionUuid, existingDirPaths, allTargetPaths, true);
        }

        // files have to be uploaded sequentially
        for (let idx = 0; idx < files.length; idx++) {
            const file = files[idx] as FileWithRelativePath;
            let nestedPath: string | undefined = undefined;

            if (isAnyNested) {
                const pathKey = getPathKey(file);

                if (file[pathKey] && file[pathKey]!.length > 0) {
                    nestedPath = file[pathKey]!.split('/').slice(0, -1).join('/');
                }
            }

            try {
                if (nestedPath) {
                    await this.uploadFile(collectionUuid, file, idx, onProgress, `${collectionUuid}/${nestedPath}/`.replace("//", "/"));
                } else {
                    await this.uploadFile(collectionUuid, file, idx, onProgress, targetLocation);
                }
            } catch (error) {
                console.error("Error uploading file", `${collectionUuid}${nestedPath ? `/${nestedPath}`: ''}/${file.name}`, error);
            }
        }
        await this.update(collectionUuid, { preserveVersion: true });
    }

    async renameFile(collectionUuid: string, collectionPdh: string, oldPath: string, newPath: string) {
        return this.replaceFiles(
            { uuid: collectionUuid },
            {
                [this.combineFilePath([newPath])]: `${collectionPdh}${this.combineFilePath([oldPath])}`,
                [this.combineFilePath([oldPath])]: "",
            }
        );
    }

    extendFileURL = (file: CollectionDirectory | CollectionFile) => {
        const baseUrl = this.keepWebdavClient.getBaseUrl().endsWith("/")
            ? this.keepWebdavClient.getBaseUrl().slice(0, -1)
            : this.keepWebdavClient.getBaseUrl();
        const apiToken = this.authService.getApiToken();
        const encodedApiToken = apiToken ? encodeURI(apiToken) : "";
        const userApiToken = `/t=${encodedApiToken}/`;
        const splittedPrevFileUrl = file.url.split("/");
        const url = `${baseUrl}/${splittedPrevFileUrl[1]}${userApiToken}${splittedPrevFileUrl.slice(2).join("/")}`;
        return {
            ...file,
            url,
        };
    };

    async getFileContents(file: CollectionFile) {
        return (await this.keepWebdavClient.get(`c=${file.id}`)).response;
    }

    private async uploadFile(
        collectionUuid: string,
        file: File,
        fileId: number,
        onProgress: UploadProgress = () => {
            return;
        },
        targetLocation: string = ""
    ) {
        const fileURL = `c=${targetLocation !== "" ? targetLocation : collectionUuid}/${file.name}`.replace("//", "/");
        const requestConfig = {
            headers: {
                "Content-Type": "text/octet-stream",
            },
            onUploadProgress: (e: ProgressEvent) => {
                onProgress(fileId, e.loaded, e.total, Date.now());
            },
        };
        return this.keepWebdavClient.upload(fileURL, [file], requestConfig);
    }

    deleteFiles(collectionUuid: string, files: string[], showErrors?: boolean) {
        const optimizedFiles = files
            .sort((a, b) => a.length - b.length)
            .reduce((acc, currentPath) => {
                const parentPathFound = acc.find(parentPath => currentPath.indexOf(`${parentPath}/`) > -1);

                if (!parentPathFound) {
                    return [...acc, currentPath];
                }

                return acc;
            }, []);

        const fileMap = optimizedFiles.reduce((obj, filePath) => {
            return {
                ...obj,
                [this.combineFilePath([filePath])]: "",
            };
        }, {});

        return this.replaceFiles({ uuid: collectionUuid }, fileMap, showErrors);
    }

    copyFiles(
        sourcePdh: string,
        files: string[],
        destinationCollection: CollectionPartialUpdateOrCreate,
        destinationPath: string,
        showErrors?: boolean
    ) {
        const fileMap = files.reduce((obj, sourceFile) => {
            const fileBasename = sourceFile.split("/").filter(Boolean).slice(-1).join("");
            return {
                ...obj,
                [this.combineFilePath([destinationPath, fileBasename])]: `${sourcePdh}${this.combineFilePath([sourceFile])}`,
            };
        }, {});

        return this.replaceFiles(destinationCollection, fileMap, showErrors);
    }

    moveFiles(
        sourceUuid: string,
        sourcePdh: string,
        files: string[],
        destinationCollection: CollectionPartialUpdateOrCreate,
        destinationPath: string,
        showErrors?: boolean
    ) {
        if (sourceUuid === destinationCollection.uuid) {
            let errors: CommonResourceServiceError[] = [];
            const fileMap = files.reduce((obj, sourceFile) => {
                const fileBasename = sourceFile.split("/").filter(Boolean).slice(-1).join("");
                const fileDestinationPath = this.combineFilePath([destinationPath, fileBasename]);
                const fileSourcePath = this.combineFilePath([sourceFile]);
                const fileSourceUri = `${sourcePdh}${fileSourcePath}`;

                if (fileDestinationPath !== fileSourcePath) {
                    return {
                        ...obj,
                        [fileDestinationPath]: fileSourceUri,
                        [fileSourcePath]: "",
                    };
                } else {
                    errors.push(CommonResourceServiceError.SOURCE_DESTINATION_CANNOT_BE_SAME);
                    return obj;
                }
            }, {});

            if (errors.length === 0) {
                return this.replaceFiles({ uuid: sourceUuid }, fileMap, showErrors);
            } else {
                return Promise.reject({ errors });
            }
        } else {
            return this.copyFiles(sourcePdh, files, destinationCollection, destinationPath, showErrors).then(() => {
                return this.deleteFiles(sourceUuid, files, showErrors);
            });
        }
    }

    createDirectory(collectionUuid: string, paths: string[], showErrors?: boolean) {
        const fileMap = paths.reduce((fMap, path)=> {
            fMap[this.combineFilePath([path])] = emptyCollectionPdh;
            return fMap;
        }, {})

        return this.replaceFiles({ uuid: collectionUuid }, fileMap, showErrors, false);
    }

    /* since creating a nested dir will create all parent dirs that don't exist,
    *  we only create the longest unique paths
    */
    async createMinNecessaryDirs(collectionUuid: string, existingDirPaths: Set<string>, targetPaths: string[], showErrors?) {
        const pathsToCreate = getMinNecessaryPaths(targetPaths).filter(path => !existingDirPaths.has(path));
        if (pathsToCreate.length > 0) {
            try {
                await this.createDirectory(collectionUuid, pathsToCreate, showErrors);
            } catch (error) {
                console.error(`Error creating directory in ${collectionUuid}`, error);
            }
        }
    }

    downloadZip(collectionUuid: string, paths: string[], fileName: string) {
        // Get webdav base url & token
        const baseUrl = this.keepWebdavClient.getBaseUrl().endsWith("/")
            ? this.keepWebdavClient.getBaseUrl().slice(0, -1)
            : this.keepWebdavClient.getBaseUrl();
        const apiToken = this.authService.getApiToken();

        // Throw error to be exposed in toast if token missing
        if (!apiToken) {
            throw new Error("Token missing");
        }

        // Create form
        const form = document.createElement("form");
        form.setAttribute("method", "get");
        form.setAttribute("action", `${baseUrl}/c=${collectionUuid}`);

        // Attach token
        const tokenInput = document.createElement("input");
        tokenInput.name = "api_token";
        tokenInput.value = apiToken;
        form.appendChild(tokenInput);

        // Add accept and disposition
        const acceptInput = document.createElement("input");
        acceptInput.name = "accept";
        acceptInput.value = "application/zip";
        form.appendChild(acceptInput);

        const dispositionInput = document.createElement("input");
        dispositionInput.name = "disposition";
        dispositionInput.value = "attachment";
        form.appendChild(dispositionInput);

        // Add filename parameter
        const fileNameInput = document.createElement("input");
        fileNameInput.name = "download_filename";
        fileNameInput.value = fileName;
        form.appendChild(fileNameInput);

        // Add file parameters for each path
        paths.forEach((path) => {
            const fileInput = document.createElement("input");
            fileInput.name = "files";
            fileInput.value = path.replace(/^\//, '');
            form.appendChild(fileInput);
        });

        // Append form to body, submit, and cleanup form
        document.body.appendChild(form);
        form.submit();
        form.remove();
    }
}


export function getMinNecessaryPaths(paths: string[]): string[] {
    //remove duplicates
    const uniquePaths = Array.from(new Set(paths)).filter(path => !!path && typeof path === 'string' && path.length > 0);

    return uniquePaths.filter((path) =>
        uniquePaths.every((existing) =>
            path === existing || !existing.startsWith(path + '/')
        )
    );

}

const getPathKey = (file: FileWithRelativePath) => {
    return file.relativePath ? 'relativePath' : 'webkitRelativePath';
}