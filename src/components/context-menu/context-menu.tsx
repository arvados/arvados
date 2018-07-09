// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from "react";
import { Popover, List, ListItem, ListItemIcon, ListItemText, Divider } from "@material-ui/core";
import { DefaultTransformOrigin } from "../popover/helpers";

export interface ContextMenuAction {
    name: string;
    icon: string;
    openCreateDialog?: boolean;
}

export type ContextMenuActionGroup = ContextMenuAction[];

export interface ContextMenuProps<T> {
    anchorEl?: HTMLElement;
    actions: ContextMenuActionGroup[];
    onActionClick: (action: ContextMenuAction) => void;
    onClose: () => void;
}

export default class ContextMenu<T> extends React.PureComponent<ContextMenuProps<T>> {
    render() {
        const { anchorEl, actions, onClose, onActionClick } = this.props;
        return <Popover
            anchorEl={anchorEl}
            open={!!anchorEl}
            onClose={onClose}
            transformOrigin={DefaultTransformOrigin}
            anchorOrigin={DefaultTransformOrigin}
            onContextMenu={this.handleContextMenu}>
            <List dense>
                {actions.map((group, groupIndex) =>
                    <React.Fragment key={groupIndex}>
                        {group.map((action, actionIndex) =>
                            <ListItem
                                button
                                key={actionIndex}
                                onClick={() => onActionClick(action)}>
                                <ListItemIcon>
                                    <i className={action.icon} />
                                </ListItemIcon>
                                <ListItemText>
                                    {action.name}
                                </ListItemText>
                            </ListItem>)}
                        {groupIndex < actions.length - 1 && <Divider />}
                    </React.Fragment>)}
            </List>
        </Popover>;
    }

    handleContextMenu = (event: React.MouseEvent<HTMLElement>) => {
        event.preventDefault();
        this.props.onClose();
    }
}
