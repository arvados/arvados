// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { OpenIcon } from "~/components/icon/icon";
import { sanitizeToken } from "./helpers";

export const FileViewerAction = (props: { href?: any, download?: any, onClick?: () => void, kind?: string, currentCollectionUuid?: string; }) => {

    return props.href
        ? <a
            style={{ textDecoration: 'none' }}
            href={sanitizeToken(props.href, true)}
            target="_blank"
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
