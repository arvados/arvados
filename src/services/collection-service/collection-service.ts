// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "~/common/api/common-resource-service";
import { CollectionResource } from "~/models/collection";
import axios, { AxiosInstance } from "axios";
import { KeepService } from "../keep-service/keep-service";
import { FilterBuilder } from "~/common/api/filter-builder";
import { CollectionFile, createCollectionFile } from "~/models/collection-file";
import { parseKeepManifestText, stringifyKeepManifest } from "../collection-files-service/collection-manifest-parser";
import * as _ from "lodash";
import { KeepManifestStream } from "~/models/keep-manifest";

export type UploadProgress = (fileId: number, loaded: number, total: number, currentTime: number) => void;

export class CollectionService extends CommonResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private keepService: KeepService) {
        super(serverApi, "collections");
    }

    private readFile(file: File): Promise<ArrayBuffer> {
        return new Promise<ArrayBuffer>(resolve => {
            const reader = new FileReader();
            reader.onload = () => {
                resolve(reader.result as ArrayBuffer);
            };

            reader.readAsArrayBuffer(file);
        });
    }

    private uploadFile(keepServiceHost: string, file: File, fileId: number, onProgress?: UploadProgress): Promise<CollectionFile> {
        return this.readFile(file).then(content => {
            return axios.post<string>(keepServiceHost, content, {
                headers: {
                    'Content-Type': 'text/octet-stream'
                },
                onUploadProgress: (e: ProgressEvent) => {
                    if (onProgress) {
                        onProgress(fileId, e.loaded, e.total, Date.now());
                    }
                    console.log(`${e.loaded} / ${e.total}`);
                }
            }).then(data => createCollectionFile({
                id: data.data,
                name: file.name,
                size: file.size
            }));
        });
    }

    private async updateManifest(collectionUuid: string, files: CollectionFile[]): Promise<CollectionResource> {
        const collection = await this.get(collectionUuid);
        const manifest: KeepManifestStream[] = parseKeepManifestText(collection.manifestText);

        files.forEach(f => {
            let kms = manifest.find(stream => stream.name === f.path);
            if (!kms) {
                kms = {
                    files: [],
                    locators: [],
                    name: f.path
                };
                manifest.push(kms);
            }
            kms.locators.push(f.id);
            const len = kms.files.length;
            const nextPos = len > 0
                ? parseInt(kms.files[len - 1].position, 10) + kms.files[len - 1].size
                : 0;
            kms.files.push({
                name: f.name,
                position: nextPos.toString(),
                size: f.size
            });
        });

        console.log(manifest);

        const manifestText = stringifyKeepManifest(manifest);
        const data = { ...collection, manifestText };
        return this.update(collectionUuid, CommonResourceService.mapKeys(_.snakeCase)(data));
    }

    uploadFiles(collectionUuid: string, files: File[], onProgress?: UploadProgress): Promise<CollectionResource | never> {
        const filters = FilterBuilder.create()
            .addEqual("service_type", "proxy");

        return this.keepService.list({ filters }).then(data => {
            if (data.items && data.items.length > 0) {
                const serviceHost =
                    (data.items[0].serviceSslFlag ? "https://" : "http://") +
                    data.items[0].serviceHost +
                    ":" + data.items[0].servicePort;

                console.log("serviceHost", serviceHost);

                const files$ = files.map((f, idx) => this.uploadFile(serviceHost, f, idx, onProgress));
                return Promise.all(files$).then(values => {
                    return this.updateManifest(collectionUuid, values);
                });
            } else {
                return Promise.reject("Missing keep service host");
            }
        });
    }
}
