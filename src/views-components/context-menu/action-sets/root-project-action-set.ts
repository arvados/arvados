// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from '../context-menu-action-set';
import { openCollectionCreateDialog } from 'store/collections/collection-create-actions';
import { NewProjectIcon, CollectionIcon } from 'components/icon/icon';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';

export const rootProjectActionSet: ContextMenuActionItemSet = [
    [
        {
            icon: NewProjectIcon,
            name: 'New project',
            execute: (dispatch, resources) => {
                 dispatch<any>(openProjectCreateDialog(resources[0].uuid));
            },
        },
        {
            icon: CollectionIcon,
            name: 'New Collection',
            execute: (dispatch, resources) => {
                 dispatch<any>(openCollectionCreateDialog(resources[0].uuid));
            },
        },
    ],
];
