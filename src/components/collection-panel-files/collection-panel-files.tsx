// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { TreeItem, TreeItemStatus } from '~/components/tree/tree';
import { FileTreeData } from '~/components/file-tree/file-tree-data';
import { FileTree } from '~/components/file-tree/file-tree';
import { IconButton, Grid, Typography, StyleRulesCallback, withStyles, WithStyles, CardHeader, Card, Button, Tooltip } from '@material-ui/core';
import { CustomizeTableIcon } from '~/components/icon/icon';
import { DownloadIcon } from '~/components/icon/icon';

export interface CollectionPanelFilesProps {
    items: Array<TreeItem<FileTreeData>>;
    isWritable: boolean;
    onUploadDataClick: () => void;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>, isWritable: boolean) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, isWritable: boolean) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
    onFileClick: (id: string) => void;
    currentItemUuid?: string;
}

type CssRules = 'root' | 'cardSubheader' | 'nameHeader' | 'fileSizeHeader' | 'uploadIcon' | 'button';

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
    },
    uploadIcon: {
        transform: 'rotate(180deg)'
    },
    button: {
        marginRight: -theme.spacing.unit,
        marginTop: '0px'
    }
});

export const CollectionPanelFiles =
    withStyles(styles)(
        ({ onItemMenuOpen, onOptionsMenuOpen, onUploadDataClick, classes, isWritable, ...treeProps }: CollectionPanelFilesProps & WithStyles<CssRules>) =>
            <Card data-cy='collection-files-panel' className={classes.root}>
                <CardHeader
                    title="Files"
                    classes={{ action: classes.button }}
                    action={
                        isWritable &&
                        <Button
                            data-cy='upload-button'
                            onClick={onUploadDataClick}
                            variant='contained'
                            color='primary'
                            size='small'>
                            <DownloadIcon className={classes.uploadIcon} />
                            Upload data
                        </Button>
                    } />
                <CardHeader
                    className={classes.cardSubheader}
                    action={
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton
                                data-cy='collection-files-panel-options-btn'
                                onClick={(ev) => onOptionsMenuOpen(ev, isWritable)}>
                                <CustomizeTableIcon />
                            </IconButton>
                        </Tooltip>
                    } />
                <Grid container justify="space-between">
                    <Typography variant="caption" className={classes.nameHeader}>
                        Name
                    </Typography>
                    <Typography variant="caption" className={classes.fileSizeHeader}>
                        File size
                    </Typography>
                </Grid>
                <FileTree onMenuOpen={(ev, item) => onItemMenuOpen(ev, item, isWritable)} {...treeProps} />
            </Card>);
