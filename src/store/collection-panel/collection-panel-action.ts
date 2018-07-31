// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";
import { collectionService } from "../../services/services";
import { collectionPanelFilesAction } from "./collection-panel-files/collection-panel-files-actions";
import { parseKeepManifestText } from "../../models/keep-manifest";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
}, { tag: 'type', value: 'payload' });

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const loadCollection = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid, kind }));
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ manifest: [] }));
        return collectionService
            .get(uuid)
            .then(item => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item }));
                const manifest = parseKeepManifestText(item.manifestText);
                dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ manifest }));
            });
    };



