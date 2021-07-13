// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { UploadFile, fileUploaderActions, FileUploaderAction } from "./file-uploader-actions";
import { uniqBy } from 'lodash';

export type UploaderState = UploadFile[];

const initialState: UploaderState = [];

export const fileUploaderReducer = (state: UploaderState = initialState, action: FileUploaderAction) => {
    return fileUploaderActions.match(action, {
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
        UPDATE_UPLOAD_FILES: files => {
            const updateFiles = files.map((f, idx) => ({
                id: state.length + idx,
                file: f,
                prevLoaded: 0,
                loaded: 0,
                total: 0,
                startTime: 0,
                prevTime: 0,
                currentTime: 0
            }));
            const updatedState = state.concat(updateFiles);
            const uniqUpdatedState = uniqBy(updatedState, 'file.name');

            return uniqUpdatedState;
        },
        DELETE_UPLOAD_FILE: file => {
            const idToDelete: number = file.id;
            const updatedState = state.filter(file => file.id !== idToDelete);

            return updatedState;
        },
        START_UPLOAD: () => {
            const startTime = Date.now();
            return state.map(f => ({ ...f, startTime, prevTime: startTime }));
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
