// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from 'redux';
import { collectionCreatorReducer, CollectionCreatorState } from "./creator/collection-creator-reducer";
import { collectionUploaderReducer, CollectionUploaderState } from "./uploader/collection-uploader-reducer";

export type CollectionsState = {
    creator: CollectionCreatorState;
    uploader: CollectionUploaderState
};

export const collectionsReducer = combineReducers({
    creator: collectionCreatorReducer,
    uploader: collectionUploaderReducer
});
