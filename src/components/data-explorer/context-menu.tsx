// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from "react";
import { Popover, List, ListItem, ListItemIcon, ListItemText, Divider } from "@material-ui/core";
import { DefaultTransformOrigin } from "../popover/helpers";

export interface ContextMenuProps {
    anchorEl?: HTMLElement;
    onClose: () => void;
}

export const ContextMenu: React.SFC<ContextMenuProps> = ({ anchorEl, onClose, children }) =>
    <Popover
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={onClose}
        transformOrigin={DefaultTransformOrigin}
        anchorOrigin={DefaultTransformOrigin}>
        <Actions />
    </Popover>;

const Actions: React.SFC = () =>
    <List dense>
        {[{
            icon: "fas fa-users",
            label: "Share"
        },
        {
            icon: "fas fa-sign-out-alt",
            label: "Move to"
        },
        {
            icon: "fas fa-star",
            label: "Add to favourite"
        },
        {
            icon: "fas fa-edit",
            label: "Rename"
        },
        {
            icon: "fas fa-copy",
            label: "Make a copy"
        },
        {
            icon: "fas fa-download",
            label: "Download"
        }].map((props, index) => <Action {...props} key={index} />)}
        < Divider />
        <Action icon="fas fa-trash-alt" label="Remove" />
    </List>;

interface ActionProps {
    icon: string;
    label: string;
}

const Action: React.SFC<ActionProps> = (props) =>
    <ListItem button>
        <ListItemIcon>
            <i className={props.icon} />
        </ListItemIcon>
        <ListItemText>
            {props.label}
        </ListItemText>
    </ListItem>;

