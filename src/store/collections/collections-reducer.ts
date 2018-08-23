// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from 'redux';
import { collectionUploaderReducer, CollectionUploaderState } from "./uploader/collection-uploader-reducer";

export type CollectionsState = {
    uploader: CollectionUploaderState
};

export const collectionsReducer = combineReducers({
    uploader: collectionUploaderReducer
});
