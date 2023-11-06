// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";

export const resourceLabel = (type: string, subtype = '') => {
    switch (type) {
        case ResourceKind.COLLECTION:
            return "Data collection";
        case ResourceKind.PROJECT:
            if (subtype === "filter") {
                return "Filter group";
            } else if (subtype === "role") {
                return "Group";
            }
            return "Project";
        case ResourceKind.PROCESS:
            return "Process";
        case ResourceKind.USER:
            return "User";
        case ResourceKind.GROUP:
            return "Group";
        case ResourceKind.VIRTUAL_MACHINE:
            return "Virtual Machine";
        case ResourceKind.WORKFLOW:
            return "Workflow";
        default:
            return "Unknown";
    }
};
