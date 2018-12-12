// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "~/models/resource";

export const resourceLabel = (type: string) => {
    switch (type) {
        case ResourceKind.COLLECTION:
            return "Data collection";
        case ResourceKind.PROJECT:
            return "Project";
        case ResourceKind.PROCESS:
            return "Process";
        case ResourceKind.USER:
            return "User";
        case ResourceKind.GROUP:
            return "Group";
        case ResourceKind.VIRTUAL_MACHINE:
            return "Virtual Machine";
        default:
            return "Unknown";
    }
};
