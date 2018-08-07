// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionUploaderAction, collectionUploaderActions, UploadFile } from "./collection-uploader-actions";

export type CollectionUploaderState = UploadFile[];

const initialState: CollectionUploaderState = [];

export const collectionUploaderReducer = (state: CollectionUploaderState = initialState, action: CollectionUploaderAction) => {
    return collectionUploaderActions.match(action, {
        SET_UPLOAD_FILES: files => files.map((f, idx) => ({
            id: idx,
            file: f,
            prevLoaded: 0,
            loaded: 0,
            total: 0,
            startTime: 0,
            prevTime: 0,
            currentTime: 0
        })),
        START_UPLOAD: () => {
            const startTime = Date.now();
            return state.map(f => ({...f, startTime, prevTime: startTime}));
        },
        SET_UPLOAD_PROGRESS: ({ fileId, loaded, total, currentTime }) =>
            state.map(f => f.id === fileId ? {
                ...f,
                prevLoaded: f.loaded,
                loaded,
                total,
                prevTime: f.currentTime,
                currentTime
            } : f),
        CLEAR_UPLOAD: () => [],
        default: () => state
    });
};
