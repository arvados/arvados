// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, CopyIcon } from "components/icon/icon";
import { openMoveCollectionDialog } from "store/collections/collection-move-actions";
import { openCollectionCopyDialog, openMultiCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from "./ms-menu-action-set";
import { TrashIcon } from "components/icon/icon";

export const msCollectionActionSet: MultiSelectMenuActionSet = [
    [
        {
            icon: CopyIcon,
            name: MultiSelectMenuActionNames.MAKE_A_COPY,
            isForMulti: true,
            execute: (dispatch, [...resources]) => {
                if (resources[0].fromContextMenu || resources.length === 1) dispatch<any>(openCollectionCopyDialog(resources[0]));
                else dispatch<any>(openMultiCollectionCopyDialog(resources[0]));
            },
        },
        {
            icon: MoveToIcon,
            name: MultiSelectMenuActionNames.MOVE_TO,
            isForMulti: true,
            execute: (dispatch, resources) => dispatch<any>(openMoveCollectionDialog(resources[0])),
        },
        {
            name: MultiSelectMenuActionNames.ADD_TO_TRASH,
            icon: TrashIcon,
            isForMulti: true,
            execute: (dispatch, resources: ContextMenuResource[]) => {
                for (const resource of [...resources]) {
                    dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
                }
            },
        },
    ],
];
