// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { CollectionFilesTree } from "../../../models/collection-file";

export const collectionPanelFilesAction = unionize({
    SET_COLLECTION_FILES: ofType<CollectionFilesTree>(),
    TOGGLE_COLLECTION_FILE_COLLAPSE: ofType<{ id: string }>(),
    TOGGLE_COLLECTION_FILE_SELECTION: ofType<{ id: string }>(),
    SELECT_ALL_COLLECTION_FILES: ofType<{}>(),
    UNSELECT_ALL_COLLECTION_FILES: ofType<{}>(),
}, { tag: 'type', value: 'payload' });

export type CollectionPanelFilesAction = UnionOf<typeof collectionPanelFilesAction>;