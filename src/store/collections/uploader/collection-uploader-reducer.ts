// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionUploaderAction, collectionUploaderActions, UploadFile } from "./collection-uploader-actions";
import * as _ from 'lodash';

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
        SET_UPLOAD_PROGRESS: ({ fileId, loaded, total, currentTime }) => {
            const files = _.cloneDeep(state);
            const f = files.find(f => f.id === fileId);
            if (f) {
                f.prevLoaded = f.loaded;
                f.loaded = loaded;
                f.total = total;
                f.prevTime = f.currentTime;
                f.currentTime = currentTime;
            }
            return files;
        },
        default: () => state
    });
};
