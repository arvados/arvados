// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { TreeItem } from "../tree/tree";
import { ProjectIcon, MoreOptionsIcon } from "../icon/icon";
import { Typography, IconButton, StyleRulesCallback, withStyles, WithStyles } from "@material-ui/core";
import { formatFileSize } from "../../common/formatters";
import { ListItemTextIcon } from "../list-item-text-icon/list-item-text-icon";
import { FileTreeData } from "./file-tree-data";

type CssRules = "root" | "spacer" | "sizeInfo" | "button";

const fileTreeItemStyle: StyleRulesCallback<CssRules> = theme => ({
    root: {
        display: "flex",
        alignItems: "center",
        paddingRight: `${theme.spacing.unit}px`
    },
    spacer: {
        flex: "1"
    },
    sizeInfo: {
        marginRight: `${theme.spacing.unit * 3}px`
    },
    button: {
        width: theme.spacing.unit * 3,
        height: theme.spacing.unit * 3
    }
});

export interface FileTreeItemProps {
    item: TreeItem<FileTreeData>;
    onMoreClick: (event: React.MouseEvent<any>, item: TreeItem<FileTreeData>) => void;
}
export const FileTreeItem = withStyles(fileTreeItemStyle)(
    class extends React.Component<FileTreeItemProps & WithStyles<CssRules>> {
        render() {
            const { classes, item } = this.props;
            return <div className={classes.root}>
                <ListItemTextIcon
                    icon={ProjectIcon}
                    name={item.data.name} />
                <div className={classes.spacer} />
                <Typography
                    className={classes.sizeInfo}
                    variant="caption">{formatFileSize(item.data.size)}</Typography>
                <IconButton
                    className={classes.button}
                    onClick={this.handleClick}>
                    <MoreOptionsIcon />
                </IconButton>
            </div >;
        }

        handleClick = (event: React.MouseEvent<any>) => {
            this.props.onMoreClick(event, this.props.item);
        }
    });

