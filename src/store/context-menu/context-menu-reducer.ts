// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "../../models/resource";
import actions, { ContextMenuAction } from "./context-menu-actions";

export interface ContextMenuState {
    position: ContextMenuPosition;
    resource?: ContextMenuResource;
}

export interface ContextMenuPosition {
    x: number;
    y: number;
}

export interface ContextMenuResource {
    uuid: string;
    kind: string;
}

const initialState = {
    position: { x: 0, y: 0 }
};

const reducer = (state: ContextMenuState = initialState, action: ContextMenuAction) =>
    actions.match(action, {
        default: () => state,
        OPEN_CONTEXT_MENU: ({resource, position}) => ({ resource, position }),
        CLOSE_CONTEXT_MENU: () => ({ position: state.position })
    });

export default reducer;
