// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "../../common/api/common-resource-service";
import { CollectionResource } from "../../models/collection";
import axios, { AxiosInstance } from "axios";
import { KeepService } from "../keep-service/keep-service";
import { FilterBuilder } from "../../common/api/filter-builder";
import { CollectionFile, CollectionFileType, createCollectionFile } from "../../models/collection-file";
import { parseKeepManifestText, stringifyKeepManifest } from "../collection-files-service/collection-manifest-parser";
import * as _ from "lodash";
import { KeepManifestStream } from "../../models/keep-manifest";

export class CollectionService extends CommonResourceService<CollectionResource> {
    constructor(serverApi: AxiosInstance, private keepService: KeepService) {
        super(serverApi, "collections");
    }

    uploadFile(keepServiceHost: string, file: File, fileIdx = 0): Promise<CollectionFile> {
        const fd = new FormData();
        fd.append(`file_${fileIdx}`, file);

        return axios.post<string>(keepServiceHost, fd, {
            onUploadProgress: (e: ProgressEvent) => {
                console.log(`${e.loaded} / ${e.total}`);
            }
        }).then(data => createCollectionFile({
            id: data.data,
            name: file.name,
            size: file.size
        }));
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

    uploadFiles(collectionUuid: string, files: File[]) {
        console.log("Uploading files", files);

        const filters = FilterBuilder.create()
            .addEqual("service_type", "proxy");

        return this.keepService.list({ filters }).then(data => {
            if (data.items && data.items.length > 0) {
                const serviceHost =
                    (data.items[0].serviceSslFlag ? "https://" : "http://") +
                    data.items[0].serviceHost +
                    ":" + data.items[0].servicePort;

                console.log("Servicehost", serviceHost);

                const files$ = files.map((f, idx) => this.uploadFile(serviceHost, f, idx));
                Promise.all(files$).then(values => {
                    this.updateManifest(collectionUuid, values).then(() => {
                        console.log("Upload done!");
                    });
                });
            }
        });
    }
}
