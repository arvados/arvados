// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import actions from "../../../store/project/project-action";
import { IconTypes } from "../../../components/icon/icon";

export const rootProjectActionSet: ContextMenuActionSet =  [[{
    icon: IconTypes.FOLDER,
    name: "New project",
    execute: (dispatch, resource) => {
        dispatch(actions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
    }
}]];