// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from 'redux';
import { collectionCreatorReducer, CollectionCreatorState } from "./creator/collection-creator-reducer";
import { collectionUpdaterReducer, CollectionUpdaterState } from "./updater/collection-updater-reducer";
import { collectionUploaderReducer, CollectionUploaderState } from "./uploader/collection-uploader-reducer";

export type CollectionsState = {
    creator: CollectionCreatorState;
    updater: CollectionUpdaterState;
    uploader: CollectionUploaderState
};

export const collectionsReducer = combineReducers({
    creator: collectionCreatorReducer,
    updater: collectionUpdaterReducer,
    uploader: collectionUploaderReducer
});
