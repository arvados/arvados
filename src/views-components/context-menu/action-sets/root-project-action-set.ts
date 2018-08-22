// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reset } from "redux-form";

import { ContextMenuActionSet } from "../context-menu-action-set";
import { openProjectCreator } from "~/store/project/project-action";
import { collectionCreateActions } from "~/store/collections/creator/collection-creator-action";
import { COLLECTION_CREATE_DIALOG } from "../../dialog-create/dialog-collection-create";
import { NewProjectIcon, CollectionIcon } from "~/components/icon/icon";

export const rootProjectActionSet: ContextMenuActionSet =  [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => dispatch<any>(openProjectCreator(resource.uuid))
    },
    {
        icon: CollectionIcon,
        name: "New Collection",
        execute: (dispatch, resource) => {
            dispatch(reset(COLLECTION_CREATE_DIALOG));
            dispatch(collectionCreateActions.OPEN_COLLECTION_CREATOR({ ownerUuid: resource.uuid }));
        }
    }
]];
