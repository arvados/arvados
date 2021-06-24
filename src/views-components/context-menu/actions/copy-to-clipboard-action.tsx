// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import * as copy from 'copy-to-clipboard';
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { Link } from "components/icon/icon";
import { getClipboardUrl } from "./helpers";

export const CopyToClipboardAction = (props: { href?: any, download?: any, onClick?: () => void, kind?: string, currentCollectionUuid?: string; }) => {
    const copyToClipboard = () => {
        if (props.href) {
            const clipboardUrl = getClipboardUrl(props.href);
            copy(clipboardUrl);
        }

        if (props.onClick) {
            props.onClick();
        }
    };

    return props.href
        ? <ListItem button onClick={copyToClipboard}>
            <ListItemIcon>
                <Link />
            </ListItemIcon>
            <ListItemText>
                Copy to clipboard
            </ListItemText>
        </ListItem>
        : null;
};