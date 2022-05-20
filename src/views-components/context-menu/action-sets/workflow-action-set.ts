// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { openRunProcess } from "store/workflow-panel/workflow-panel-actions";

export const workflowActionSet: ContextMenuActionSet = [[
    {
        name: "Run",
        execute: (dispatch, resource) => {
            dispatch<any>(openRunProcess(resource.uuid, resource.ownerUuid, resource.name));
        }
    },
]];
