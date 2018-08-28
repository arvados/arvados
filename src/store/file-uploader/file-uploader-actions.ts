// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

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

export const fileUploaderActions = unionize({
    CLEAR_UPLOAD: ofType(),
    SET_UPLOAD_FILES: ofType<File[]>(),
    SET_UPLOAD_PROGRESS: ofType<{ fileId: number, loaded: number, total: number, currentTime: number }>(),
    START_UPLOAD: ofType(),
});

export type FileUploaderAction = UnionOf<typeof fileUploaderActions>;
