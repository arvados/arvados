// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface RuntimeStatus {
    error?: string;
    warning?: string;
    activity?: string;
    errorDetail?: string;
    warningDetail?: string;
}
