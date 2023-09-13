// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { MoveToIcon, CopyIcon } from "components/icon/icon";
import { openMoveCollectionDialog } from "store/collections/collection-move-actions";
import { openCollectionCopyDialog, openMultiCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";

export const msCollectionActionSet: ContextMenuActionSet = [
    [
        {
            icon: CopyIcon,
            name: "Make a copy",
            execute: (dispatch, resources) => {
                if (resources[0].isSingle || resources.length === 1) dispatch<any>(openCollectionCopyDialog(resources[0]));
                else dispatch<any>(openMultiCollectionCopyDialog(resources[0]));
            },
        },
        {
            icon: MoveToIcon,
            name: "Move to",
            execute: (dispatch, resources) => dispatch<any>(openMoveCollectionDialog(resources[0])),
        },
        {
            component: ToggleTrashAction,
            name: "ToggleTrashAction",
            execute: (dispatch, resources: ContextMenuResource[]) => {
                for (const resource of resources) {
                    dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
                }
            },
        },
    ],
];
