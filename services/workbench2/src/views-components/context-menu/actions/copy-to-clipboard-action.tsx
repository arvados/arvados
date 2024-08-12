// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import copy from 'copy-to-clipboard';
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { Link } from "components/icon/icon";

interface CopyToClipboardActionProps {
    href?: any;
    kind?: string;
    customText?: string;
    onClick?: () => void;
};

export const CopyToClipboardAction = (props: CopyToClipboardActionProps) => {
    const copyToClipboard = () => {
        if (props.href) {
	    copy(props.href);
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
                 {props.customText ? props.customText : "Copy link to clipboard"}
             </ListItemText>
         </ListItem>
         : null;
};
