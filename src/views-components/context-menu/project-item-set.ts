// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuItemGroup } from "../../components/context-menu/context-menu";
import { ContextMenuItemSet } from "./context-menu-item-set";
import actions from "../../store/project/project-action";

export const projectItemSet: ContextMenuItemSet = {
    getItems: () => items,
    handleItem: (dispatch, item, resource) => {
        if (item.name === "New project") {
            dispatch(actions.OPEN_PROJECT_CREATOR({ ownerUuid: resource.uuid }));
        }
    }
};

const items: ContextMenuItemGroup[] = [[{
    icon: "fas fa-plus fa-fw",
    name: "New project"
}, {
    icon: "fas fa-users fa-fw",
    name: "Share"
}, {
    icon: "fas fa-sign-out-alt fa-fw",
    name: "Move to"
}, {
    icon: "fas fa-star fa-fw",
    name: "Add to favourite"
}, {
    icon: "fas fa-edit fa-fw",
    name: "Rename"
}, {
    icon: "fas fa-copy fa-fw",
    name: "Make a copy"
}, {
    icon: "fas fa-download fa-fw",
    name: "Download"
}], [{
    icon: "fas fa-trash-alt fa-fw",
    name: "Remove"
}
]];