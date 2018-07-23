// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from "react";
import { Popover, List, ListItem, ListItemIcon, ListItemText, Divider } from "@material-ui/core";
import { DefaultTransformOrigin } from "../popover/helpers";
import { IconType } from "../icon/icon";

export interface ContextMenuItem {
    name: string;
    icon: IconType;
}

export type ContextMenuItemGroup = ContextMenuItem[];

export interface ContextMenuProps {
    anchorEl?: HTMLElement;
    items: ContextMenuItemGroup[];
    onItemClick: (action: ContextMenuItem) => void;
    onClose: () => void;
}

export class ContextMenu extends React.PureComponent<ContextMenuProps> {
    render() {
        const { anchorEl, items, onClose, onItemClick} = this.props;
        return <Popover
            anchorEl={anchorEl}
            open={!!anchorEl}
            onClose={onClose}
            transformOrigin={DefaultTransformOrigin}
            anchorOrigin={DefaultTransformOrigin}
            onContextMenu={this.handleContextMenu}>
            <List dense>
                {items.map((group, groupIndex) =>
                    <React.Fragment key={groupIndex}>
                        {group.map((item, actionIndex) =>
                            <ListItem
                                button
                                key={actionIndex}
                                onClick={() => onItemClick(item)}>
                                <ListItemIcon>
                                    <item.icon/>
                                </ListItemIcon>
                                <ListItemText>
                                    {item.name}
                                </ListItemText>
                            </ListItem>)}
                        {groupIndex < items.length - 1 && <Divider />}
                    </React.Fragment>)}
            </List>
        </Popover>;
    }

    handleContextMenu = (event: React.MouseEvent<HTMLElement>) => {
        event.preventDefault();
        this.props.onClose();
    }
}
