// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from '../context-menu-action-set';
import { ToggleTrashAction } from 'views-components/context-menu/actions/trash-action';
import { openTrashConfirmDialog } from 'store/trash/trash-actions';

export const trashActionSet: ContextMenuActionSet = [
    [
        {
            component: ToggleTrashAction,
            name: ContextMenuActionNames.MOVE_TO_TRASH,
            execute: (dispatch, resources) => {
                dispatch<any>(openTrashConfirmDialog(resources.map(res => res.uuid), resources.some(res => res.isTrashed)));
            },
        },
    ],
];
