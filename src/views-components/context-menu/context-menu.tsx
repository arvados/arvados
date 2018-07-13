// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect, Dispatch, DispatchProp } from "react-redux";
import { RootState } from "../../store/store";
import actions from "../../store/context-menu/context-menu-actions";
import ContextMenu, { ContextMenuProps, ContextMenuItem } from "../../components/context-menu/context-menu";
import { createAnchorAt } from "../../components/popover/helpers";
import { ContextMenuResource } from "../../store/context-menu/context-menu-reducer";
import { ContextMenuItemSet } from "./context-menu-item-set";
import { emptyItemSet } from "./item-sets/empty-item-set";

type DataProps = Pick<ContextMenuProps, "anchorEl" | "items"> & { resource?: ContextMenuResource };
const mapStateToProps = (state: RootState): DataProps => {
    const { position, resource } = state.contextMenu;
    return {
        anchorEl: resource ? createAnchorAt(position) : undefined,
        items: getMenuItemSet(resource).getItems(),
        resource
    };
};

type ActionProps = Pick<ContextMenuProps, "onClose"> & { onItemClick: (item: ContextMenuItem, resource?: ContextMenuResource) => void };
const mapDispatchToProps = (dispatch: Dispatch): ActionProps => ({
    onClose: () => {
        dispatch(actions.CLOSE_CONTEXT_MENU());
    },
    onItemClick: (item: ContextMenuItem, resource?: ContextMenuResource) => {
        dispatch(actions.CLOSE_CONTEXT_MENU());
        if (resource) {
            getMenuItemSet(resource).handleItem(dispatch, item, resource);
        }
    }
});

const mergeProps = ({ resource, ...dataProps }: DataProps, actionProps: ActionProps): ContextMenuProps => ({
    ...dataProps,
    ...actionProps,
    onItemClick: item => {
        actionProps.onItemClick(item, resource);
    }
});

export const ContextMenuHOC = connect(mapStateToProps, mapDispatchToProps, mergeProps)(ContextMenu);

const menuItemSets = new Map<string, ContextMenuItemSet>();

export const addMenuItemsSet = (name: string, itemSet: ContextMenuItemSet) => {
    menuItemSets.set(name, itemSet);
};

const getMenuItemSet = (resource?: ContextMenuResource): ContextMenuItemSet => {
    return resource ? menuItemSets.get(resource.kind) || emptyItemSet : emptyItemSet;
};

