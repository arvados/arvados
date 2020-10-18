// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { connect } from 'react-redux';
import { ListItemIcon, ListItemText, ListItem } from "@material-ui/core";
import { OpenIcon } from "~/components/icon/icon";
import { sanitizeToken } from "./helpers";
import { RootState } from "~/store/store";

export const FileViewerAction = (props: any) => {
    const {
        keepWebServiceUrl,
        keepWebInlineServiceUrl,
    } = props;

    return props.href
        ? <a
            style={{ textDecoration: 'none' }}
            href={sanitizeToken(props.href.replace(keepWebServiceUrl, keepWebInlineServiceUrl), true)}
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

const mapStateToProps = ({ auth }: RootState): any => ({
    keepWebServiceUrl: auth.config.keepWebServiceUrl,
    keepWebInlineServiceUrl: auth.config.keepWebInlineServiceUrl,
});


export default connect(mapStateToProps, null)(FileViewerAction);
