// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from 'redux';
import * as creator from "./creator/collection-creator-reducer";
import * as updator from "./updater/collection-updater-reducer";

export type CollectionsState = {
    creator: creator.CollectionCreatorState;
    updator: updator.CollectionUpdatorState;
};

export const collectionsReducer = combineReducers({
    creator: creator.collectionCreationReducer,
    updator: updator.collectionCreationReducer
});
