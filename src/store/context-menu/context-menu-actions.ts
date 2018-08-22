// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { ContextMenuPosition, ContextMenuResource } from "./context-menu-reducer";
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { Dispatch } from 'redux';

export const contextMenuActions = unionize({
    OPEN_CONTEXT_MENU: ofType<{ position: ContextMenuPosition, resource: ContextMenuResource }>(),
    CLOSE_CONTEXT_MENU: ofType<{}>()
});

export type ContextMenuAction = UnionOf<typeof contextMenuActions>;

export const openContextMenu = (event: React.MouseEvent<HTMLElement>, resource: { name: string; uuid: string; description?: string; kind: ContextMenuKind; }) =>
    (dispatch: Dispatch) => {
        event.preventDefault();
        dispatch(
            contextMenuActions.OPEN_CONTEXT_MENU({
                position: { x: event.clientX, y: event.clientY },
                resource
            })
        );
    };