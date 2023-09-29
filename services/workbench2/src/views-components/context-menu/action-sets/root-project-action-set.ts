// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { openCollectionCreateDialog } from 'store/collections/collection-create-actions';
import { NewProjectIcon, CollectionIcon } from "components/icon/icon";
import { openProjectCreateDialog } from 'store/projects/project-create-actions';

export const rootProjectActionSet: ContextMenuActionSet =  [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch<any>(openProjectCreateDialog(resource.uuid));
        }
    },
    {
        icon: CollectionIcon,
        name: "New Collection",
        execute: (dispatch, resource) => {
            dispatch<any>(openCollectionCreateDialog(resource.uuid));
        }
    }
]];
