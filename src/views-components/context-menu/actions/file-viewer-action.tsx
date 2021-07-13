// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { OpenIcon } from "components/icon/icon";

export const FileViewerAction = (props: any) => {
    return props.href
        ? <a
            style={{ textDecoration: 'none' }}
            href={props.href}
            target="_blank"
            rel="noopener noreferrer"
            onClick={props.onClick}>
            <ListItem button>
                <ListItemIcon>
                    <OpenIcon />
                </ListItemIcon>
                <ListItemText>
                    Open in new tab
                    </ListItemText>
            </ListItem>
        </a>
        : null;
};
