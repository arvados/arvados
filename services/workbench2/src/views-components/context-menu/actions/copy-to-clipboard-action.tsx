// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import copy from 'copy-to-clipboard';
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { Link } from "components/icon/icon";
import { getCollectionItemClipboardUrl } from "./helpers";

export const CopyToClipboardAction = (props: { href?: any, download?: any, onClick?: () => void, kind?: string, currentCollectionUuid?: string; }) => {
    const copyToClipboard = () => {
        if (props.href) {
            const clipboardUrl = getCollectionItemClipboardUrl(props.href, true, true);
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
