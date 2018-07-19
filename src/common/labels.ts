// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "../models/resource";

export const RESOURCE_LABEL = (type: string) => {
    switch (type) {
        case ResourceKind.Collection:
            return "Data collection";
        case ResourceKind.Project:
            return "Project";
        case ResourceKind.Process:
            return "Process";
        default:
            return "Unknown";
    }
};