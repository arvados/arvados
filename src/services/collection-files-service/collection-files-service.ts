// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionService } from "../collection-service/collection-service";
import { parseKeepManifestText, stringifyKeepManifest } from "./collection-manifest-parser";
import { mapManifestToCollectionFilesTree } from "./collection-manifest-mapper";
import { CollectionFile } from "../../models/collection-file";
import { CommonResourceService } from "../../common/api/common-resource-service";
import * as _ from "lodash";

export class CollectionFilesService {

    constructor(private collectionService: CollectionService) { }

    getFiles(collectionUuid: string) {
        return this.collectionService
            .get(collectionUuid)
            .then(collection =>
                mapManifestToCollectionFilesTree(
                    parseKeepManifestText(
                        collection.manifestText
                    )
                )
            );
    }

    async renameFile(collectionUuid: string, file: { name: string, path: string }, newName: string) {
        const collection = await this.collectionService.get(collectionUuid);
        const manifest = parseKeepManifestText(collection.manifestText);
        const updatedManifest = manifest.map(
            stream => stream.name === file.path
                ? {
                    ...stream,
                    files: stream.files.map(
                        f => f.name === file.name
                            ? { ...f, name: newName }
                            : f
                    )
                }
                : stream
        );
        const manifestText = stringifyKeepManifest(updatedManifest);
        const data = { ...collection, manifestText };
        return this.collectionService.update(collectionUuid, CommonResourceService.mapKeys(_.snakeCase)(data));
    }

    async deleteFile(collectionUuid: string, file: { name: string, path: string }) {
        const collection = await this.collectionService.get(collectionUuid);
        const manifest = parseKeepManifestText(collection.manifestText);
        const updatedManifest = manifest.map(stream =>
            stream.name === file.path
                ? {
                    ...stream,
                    files: stream.files.filter(f => f.name !== file.name)
                }
                : stream
        );
        const manifestText = stringifyKeepManifest(updatedManifest);
        return this.collectionService.update(collectionUuid, { manifestText });
    }

    renameTest() {
        const u = this.renameFile('qr1hi-4zz18-n0sx074erl4p0ph', {
            name: 'extracted2.txt.png',
            path: ''
        }, 'extracted-new.txt.png');
    }
}
