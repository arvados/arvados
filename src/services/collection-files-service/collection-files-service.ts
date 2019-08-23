// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionService } from "../collection-service/collection-service";
import { parseKeepManifestText, stringifyKeepManifest } from "./collection-manifest-parser";
import { mapManifestToCollectionFilesTree } from "./collection-manifest-mapper";

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
        return this.collectionService.update(collectionUuid, { manifestText });
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
}
