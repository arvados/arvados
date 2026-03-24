// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isFileOperationLocation } from "store/tree-picker/tree-picker-actions";

export const isValidFileOpsLocation = (value: any) => {
    if (isFileOperationLocation(value)) {
        return undefined;
    }
    return 'Invalid file operation location.';
}