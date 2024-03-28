// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from '../context-menu-action-set';
import { openCollectionCreateDialog } from 'store/collections/collection-create-actions';
import { NewProjectIcon, CollectionIcon } from 'components/icon/icon';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';

export const rootProjectActionSet: ContextMenuActionSet = [
    [
        {
            icon: NewProjectIcon,
            name: ContextMenuActionNames.NEW_PROJECT,
            execute: (dispatch, resources) => {
                 dispatch<any>(openProjectCreateDialog(resources[0].uuid));
            },
        },
        {
            icon: CollectionIcon,
            name: ContextMenuActionNames.NEW_COLLECTION,
            execute: (dispatch, resources) => {
                 dispatch<any>(openCollectionCreateDialog(resources[0].uuid));
            },
        },
    ],
];
