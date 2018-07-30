// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { FileTreeData } from '../file-tree/file-tree-data';
import { FileTree } from '../file-tree/file-tree';
import { IconButton, Grid, Typography, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import { CustomizeTableIcon } from '../icon/icon';

export interface CollectionPanelFilesProps {
    items: Array<TreeItem<FileTreeData>>;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
}

type CssRules = 'nameHeader' | 'fileSizeHeader';

const styles: StyleRulesCallback<CssRules> = theme => ({
    nameHeader: {
        marginLeft: '75px'
    },
    fileSizeHeader: {
        marginRight: '50px'
    }
});

export const CollectionPanelFiles = withStyles(styles)(
    ({ onItemMenuOpen, onOptionsMenuOpen, classes, ...treeProps }: CollectionPanelFilesProps & WithStyles<CssRules>) =>
        <div>
            <Grid container justify="flex-end">
                <IconButton onClick={onOptionsMenuOpen}>
                    <CustomizeTableIcon />
                </IconButton>
            </Grid>
            <Grid container justify="space-between">
                <Typography variant="caption" className={classes.nameHeader}>
                    Name
                </Typography>
                <Typography variant="caption" className={classes.fileSizeHeader}>
                    File size
            </Typography>
            </Grid>
            <FileTree onMenuOpen={onItemMenuOpen} {...treeProps} />
        </div>);

