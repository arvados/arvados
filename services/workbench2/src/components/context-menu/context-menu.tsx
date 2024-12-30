// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import React from "react";
import { Popover, List, ListItem, ListItemIcon, ListItemText, Divider } from "@mui/material";
import { DefaultTransformOrigin, createAnchorAt } from "../popover/helpers";
import { IconType } from "../icon/icon";
import { RootState } from "store/store";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { sortMenuItems, ContextMenuKind, menuDirection } from "views-components/context-menu/menu-item-sort";
import { ContextMenuState } from "store/context-menu/context-menu-reducer";

export interface ContextMenuItem {
    name?: string | React.ComponentType;
    icon?: IconType;
    component?: React.ComponentType<any>;
    adminOnly?: boolean;
    filters?: ((state: RootState, resource: ContextMenuResource) => boolean)[]
}

export type ContextMenuItemGroup = ContextMenuItem[];

export interface ContextMenuProps {
    items: ContextMenuActionSet;
    contextMenu: ContextMenuState;
    onItemClick: (action: ContextMenuItem, resource: ContextMenuResource | undefined) => void;
    onClose: () => void;
}
export class ContextMenu extends React.PureComponent<ContextMenuProps> {
    render() {
        const { items, onClose, onItemClick } = this.props;
        const { open, position, resource } = this.props.contextMenu;
        const anchorEl = resource ? createAnchorAt(position) : undefined;
        return <Popover
            anchorEl={anchorEl}
            open={open}
            onClose={onClose}
            transformOrigin={DefaultTransformOrigin}
            anchorOrigin={DefaultTransformOrigin}
            onContextMenu={this.handleContextMenu}>
            <List data-cy='context-menu' dense>
                {items.map((group, groupIndex) =>
                    <React.Fragment key={groupIndex}>
                        {group.map((item, actionIndex) =>
                            item.component
                                ? <item.component
                                    key={actionIndex}
                                    data-cy={item.name}
                                    onClick={() => onItemClick(item, resource)} />
                                : <ListItem
                                    button
                                    key={actionIndex}
                                    data-cy={item.name}
                                    onClick={() => onItemClick(item, resource)}>
                                    {item.icon &&
                                        <ListItemIcon>
                                            <item.icon />
                                        </ListItemIcon>}
                                    {item.name &&
                                        <ListItemText>
                                            {item.name}
                                        </ListItemText>}
                                </ListItem>)}
                        {
                            items[groupIndex + 1] &&
                            items[groupIndex + 1].length > 0 &&
                            <Divider />
                        }
                    </React.Fragment>)}
            </List>
        </Popover>;
    }

    handleContextMenu = (event: React.MouseEvent<HTMLElement>) => {
        event.preventDefault();
        this.props.onClose();
    }
}

const menuActionSets = new Map<string, ContextMenuActionSet>();

export const addMenuActionSet = (name: ContextMenuKind, itemSet: ContextMenuActionSet) => {
    const sorted = itemSet.map(items => sortMenuItems(name, items, menuDirection.VERTICAL));
    menuActionSets.set(name, sorted);
};
const emptyActionSet: ContextMenuActionSet = [];

export const getMenuActionSet = (resource?: ContextMenuResource): ContextMenuActionSet =>
    resource ? menuActionSets.get(resource.menuKind) || emptyActionSet : emptyActionSet;
