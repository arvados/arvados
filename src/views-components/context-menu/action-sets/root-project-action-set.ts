// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reset } from "redux-form";

import { ContextMenuActionSet } from "../context-menu-action-set";
import { projectActions } from "~/store/project/project-action";
import { PROJECT_CREATE_DIALOG } from "../../dialog-create/dialog-project-create";
import { COLLECTION_CREATE_FORM_NAME, openCollectionCreateDialog } from '~/store/collections/collection-create-actions';
import { NewProjectIcon, CollectionIcon } from "~/components/icon/icon";

export const rootProjectActionSet: ContextMenuActionSet =  [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch(reset(PROJECT_CREATE_DIALOG));
            dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
        }
    },
    {
        icon: CollectionIcon,
        name: "New Collection",
        execute: (dispatch, resource) => {
            dispatch(reset(COLLECTION_CREATE_FORM_NAME));
            dispatch<any>(openCollectionCreateDialog());
        }
    }
]];
