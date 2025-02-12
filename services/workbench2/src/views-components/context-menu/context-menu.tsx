// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "store/store";
import { contextMenuActions, ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenu as ContextMenuComponent, ContextMenuProps, ContextMenuItem } from "components/context-menu/context-menu";
import { createAnchorAt } from "components/popover/helpers";
import { ContextMenuActionSet, ContextMenuAction } from "./context-menu-action-set";
import { Dispatch } from "redux";
import { memoize } from "lodash";
import { sortMenuItems, ContextMenuKind, menuDirection } from "./menu-item-sort";

type DataProps = Pick<ContextMenuProps, "anchorEl" | "items" | "open"> & { resource?: ContextMenuResource };

const mapStateToProps = (state: RootState): DataProps => {
    const { open, position, resource } = state.contextMenu;
    const filteredItems = getMenuActionSet(resource).map(group =>
        group.filter(item => {
            if (resource && item.filters) {
                // Execute all filters on this item, every returns true IFF all filters return true
                return item.filters.every(filter => filter(state, resource));
            } else {
                return true;
            }
        })
    );

    return {
        anchorEl: resource ? createAnchorAt(position) : undefined,
        items: filteredItems,
        open,
        resource,
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

const handleItemClick = memoize(
    (resource: DataProps["resource"], onItemClick: ActionProps["onItemClick"]): ContextMenuProps["onItemClick"] =>
        item => {
            onItemClick(item, { ...resource, fromContextMenu: true } as ContextMenuResource);
        }
);

const mergeProps = ({ resource, ...dataProps }: DataProps, actionProps: ActionProps): ContextMenuProps => ({
    ...dataProps,
    ...actionProps,
    onItemClick: handleItemClick(resource, actionProps.onItemClick),
});

export const ContextMenu = connect(mapStateToProps, mapDispatchToProps, mergeProps)(ContextMenuComponent);

const menuActionSets = new Map<string, ContextMenuActionSet>();

export const addMenuActionSet = (name: ContextMenuKind, itemSet: ContextMenuActionSet) => {
    const sorted = itemSet.map(items => sortMenuItems(name, items, menuDirection.VERTICAL));
    menuActionSets.set(name, sorted);
};

const emptyActionSet: ContextMenuActionSet = [];
const getMenuActionSet = (resource?: ContextMenuResource): ContextMenuActionSet =>
    resource ? menuActionSets.get(resource.menuKind) || emptyActionSet : emptyActionSet;

export const getMenuActionSetByKind = (kind: ContextMenuKind): ContextMenuActionSet =>
    menuActionSets.get(kind) || emptyActionSet;