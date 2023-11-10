// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from '../context-menu-action-set';
import { ToggleTrashAction } from 'views-components/context-menu/actions/trash-action';
import { toggleTrashed } from 'store/trash/trash-actions';

export const trashActionSet: ContextMenuActionItemSet = [
    [
        {
            component: ToggleTrashAction,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(toggleTrashed(resource.kind, resource.uuid, resource.ownerUuid, resource.isTrashed!!)));
            },
        },
    ],
];
