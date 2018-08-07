// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionService } from "../collection-service/collection-service";
import { parseKeepManifestText } from "./collection-manifest-parser";
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

}