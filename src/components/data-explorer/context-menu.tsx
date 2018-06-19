// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from "react";
import { Popover, List, ListItem, ListItemIcon, ListItemText, Divider } from "@material-ui/core";
import { DefaultTransformOrigin } from "../popover/helpers";
import { DataItem } from "./data-item";

export type ContextMenuAction = (item: DataItem) => void;
export interface ContextMenuActions {
    onShare: ContextMenuAction;
    onMoveTo: ContextMenuAction;
    onAddToFavourite: ContextMenuAction;
    onRename: ContextMenuAction;
    onCopy: ContextMenuAction;
    onDownload: ContextMenuAction;
    onRemove: ContextMenuAction;
}
export interface ContextMenuProps {
    anchorEl?: HTMLElement;
    item?: DataItem;
    onClose: () => void;
    actions: ContextMenuActions;
}

export const ContextMenu: React.SFC<ContextMenuProps> = ({ anchorEl, onClose, actions, item }) =>
    <Popover
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={onClose}
        transformOrigin={DefaultTransformOrigin}
        anchorOrigin={DefaultTransformOrigin}>
        <Actions {...{ actions, item, onClose }} />
    </Popover>;

interface ActionsProps {
    actions: ContextMenuActions;
    item?: DataItem;
    onClose: () => void;
}

const Actions: React.SFC<ActionsProps> = ({ actions, item, onClose }) =>
    <List dense>
        {[{
            icon: "fas fa-users",
            label: "Share",
            onClick: actions.onShare
        },
        {
            icon: "fas fa-sign-out-alt",
            label: "Move to",
            onClick: actions.onMoveTo
        },
        {
            icon: "fas fa-star",
            label: "Add to favourite",
            onClick: actions.onAddToFavourite
        },
        {
            icon: "fas fa-edit",
            label: "Rename",
            onClick: actions.onRename
        },
        {
            icon: "fas fa-copy",
            label: "Make a copy",
            onClick: actions.onCopy
        },
        {
            icon: "fas fa-download",
            label: "Download",
            onClick: actions.onDownload
        }].map((props, index) =>
            <Action
                item={item}
                onClose={onClose}
                key={index}
                {...props} />)}
        < Divider />
        <Action
            icon="fas fa-trash-alt"
            label="Remove"
            item={item}
            onClose={onClose}
            onClick={actions.onRemove} />
    </List>;

interface ActionProps {
    onClick: ContextMenuAction;
    item?: DataItem;
    icon: string;
    label: string;
    onClose: () => void;
}

const Action: React.SFC<ActionProps> = ({ onClick, onClose, item, icon, label }) =>
    <ListItem button onClick={() => {
        if (item) {
            onClick(item);
            onClose();
        }
    }}>
        <ListItemIcon>
            <i className={icon} />
        </ListItemIcon>
        <ListItemText>
            {label}
        </ListItemText>
    </ListItem >;

