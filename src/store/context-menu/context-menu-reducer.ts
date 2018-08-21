// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { contextMenuActions, ContextMenuAction } from "./context-menu-actions";

export interface ContextMenuState {
    open: boolean;
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
    name: string;
    description?: string;
    isTrashed?: boolean;
    ownerUuid?: string;
}

const initialState = {
    open: false,
    position: { x: 0, y: 0 }
};

export const contextMenuReducer = (state: ContextMenuState = initialState, action: ContextMenuAction) =>
    contextMenuActions.match(action, {
        default: () => state,
        OPEN_CONTEXT_MENU: ({ resource, position }) => ({ open: true, resource, position }),
        CLOSE_CONTEXT_MENU: () => ({ ...state, open: false })
    });

