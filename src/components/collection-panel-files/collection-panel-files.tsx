// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { TreeItem, TreeItemStatus } from '../tree/tree';
import { FileTreeData } from '../file-tree/file-tree-data';
import { FileTree } from '../file-tree/file-tree';
import { IconButton, Grid, Typography, StyleRulesCallback, withStyles, WithStyles, CardHeader, CardContent, Card, Button } from '@material-ui/core';
import { CustomizeTableIcon } from '../icon/icon';

export interface CollectionPanelFilesProps {
    items: Array<TreeItem<FileTreeData>>;
    onUploadDataClick: () => void;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
}

type CssRules = 'root' | 'cardSubheader' | 'nameHeader' | 'fileSizeHeader';

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
        paddingBottom: theme.spacing.unit
    },
    cardSubheader: {
        paddingTop: 0,
        paddingBottom: 0
    },
    nameHeader: {
        marginLeft: '75px'
    },
    fileSizeHeader: {
        marginRight: '65px'
    }
});

export const CollectionPanelFiles = withStyles(styles)(
    ({ onItemMenuOpen, onOptionsMenuOpen, classes, ...treeProps }: CollectionPanelFilesProps & WithStyles<CssRules>) =>
        <Card className={classes.root}>
            <CardHeader
                title="Files"
                action={
                    <Button 
                        variant='raised' 
                        color='primary'
                        size='small'>
                        Upload data
                    </Button>
                } />
            <CardHeader
                className={classes.cardSubheader}
                action={
                    <IconButton onClick={onOptionsMenuOpen}>
                        <CustomizeTableIcon />
                    </IconButton>
                } />
            <Grid container justify="space-between">
                <Typography variant="caption" className={classes.nameHeader}>
                    Name
                    </Typography>
                <Typography variant="caption" className={classes.fileSizeHeader}>
                    File size
                    </Typography>
            </Grid>
            <FileTree onMenuOpen={onItemMenuOpen} {...treeProps} />
        </Card>);
