// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionUploaderAction, collectionUploaderActions } from "./collection-uploader-actions";
import { CollectionUploadFile } from "../../../models/collection-file";

export interface CollectionUploaderState {
    files: File[];
}

const initialState: CollectionUploaderState = {
    files: []
};

export const collectionUploaderReducer = (state: CollectionUploaderState = initialState, action: CollectionUploaderAction) => {
    return collectionUploaderActions.match(action, {
        SET_UPLOAD_FILES: (files) => ({
            ...state,
            files
        }),
        default: () => state
    });
};
