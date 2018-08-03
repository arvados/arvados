// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";
import { collectionService, collectionFilesService } from "../../services/services";
import { collectionPanelFilesAction } from "./collection-panel-files/collection-panel-files-actions";
import { createTree } from "../../models/tree";
import { mapManifestToCollectionFilesTree } from "../../services/collection-files-service/collection-manifest-mapper";
import { parseKeepManifestText } from "../../services/collection-files-service/collection-manifest-parser";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
}, { tag: 'type', value: 'payload' });

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const loadCollection = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid, kind }));
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ files: createTree() }));
        return collectionService
            .get(uuid)
            .then(item => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item }));
                return collectionFilesService.getFiles(item.uuid);
            })
            .then(files => {
                dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ files }));
            });
    };



