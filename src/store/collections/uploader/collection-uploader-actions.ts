// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

export const collectionUploaderActions = unionize({
    SET_UPLOAD_FILES: ofType<File[]>(),
    START_UPLOADING: ofType<{}>(),
    UPDATE_UPLOAD_PROGRESS: ofType<{}>()
}, {
    tag: 'type',
    value: 'payload'
});

export type CollectionUploaderAction = UnionOf<typeof collectionUploaderActions>;
