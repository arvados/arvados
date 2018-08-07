// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

export interface UploadFile {
    id: number;
    file: File;
    prevLoaded: number;
    loaded: number;
    total: number;
    startTime: number;
    prevTime: number;
    currentTime: number;
}

export const collectionUploaderActions = unionize({
    SET_UPLOAD_FILES: ofType<File[]>(),
    START_UPLOAD: ofType(),
    SET_UPLOAD_PROGRESS: ofType<{ fileId: number, loaded: number, total: number, currentTime: number }>(),
    CLEAR_UPLOAD: ofType()
}, {
    tag: 'type',
    value: 'payload'
});

export type CollectionUploaderAction = UnionOf<typeof collectionUploaderActions>;
