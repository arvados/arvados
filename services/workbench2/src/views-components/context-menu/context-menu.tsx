// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { contextMenuActions, ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenu as ContextMenuComponent, ContextMenuProps, ContextMenuItem } from "components/context-menu/context-menu";
import { ContextMenuAction } from "./context-menu-action-set";
import { Dispatch } from "redux";
import { memoize } from "lodash";
import { getMenuActionSet } from "common/menu-action-set-actions";

type DataProps = Pick<ContextMenuProps, "contextMenu" | "items"> & { resource?: ContextMenuResource };

const filteredItems = memoize((resource: ContextMenuResource | undefined, state: RootState) => {
    const actionSet = getMenuActionSet(resource);
    return actionSet.map(group => group.filter(action => {
        if(resource && action.filters) {
            return action.filters.every(filter => filter(state, resource))
        } else {
            return true;
        }
    }));
});

const mapStateToProps = (state: RootState): DataProps => {
    return {
        items: filteredItems(state.contextMenu.resource, state),
        contextMenu: state.contextMenu,
    };
};

type ActionProps = Pick<ContextMenuProps, "onClose"> & { onItemClick: (item: ContextMenuItem, resource?: ContextMenuResource) => void };
const mapDispatchToProps = (dispatch: Dispatch): ActionProps => ({
    onClose: () => {
        dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
    },
    onItemClick: (action: ContextMenuAction, resource?: ContextMenuResource) => {
        dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
        if (resource) {
            action.execute(dispatch, [resource]);
        }
    },
});

export const ContextMenu = connect(mapStateToProps, mapDispatchToProps)(ContextMenuComponent);




