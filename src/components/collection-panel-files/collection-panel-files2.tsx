// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { TreeItem, TreeItemStatus } from 'components/tree/tree';
import { FileTreeData } from 'components/file-tree/file-tree-data';
import { FileTree } from 'components/file-tree/file-tree';
import { IconButton, Grid, Typography, StyleRulesCallback, withStyles, WithStyles, CardHeader, Card, Button, Tooltip, CircularProgress } from '@material-ui/core';
import { CustomizeTableIcon } from 'components/icon/icon';
import { DownloadIcon } from 'components/icon/icon';
import { SearchInput } from '../search-input/search-input';

export interface CollectionPanelFilesProps {
    items: Array<TreeItem<FileTreeData>>;
    isWritable: boolean;
    isLoading: boolean;
    tooManyFiles: boolean;
    onUploadDataClick: () => void;
    onSearchChange: (searchValue: string) => void;
    onItemMenuOpen: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>, isWritable: boolean) => void;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, isWritable: boolean) => void;
    onSelectionToggle: (event: React.MouseEvent<HTMLElement>, item: TreeItem<FileTreeData>) => void;
    onCollapseToggle: (id: string, status: TreeItemStatus) => void;
    onFileClick: (id: string) => void;
    loadFilesFunc: () => void;
    currentItemUuid?: string;
}

export type CssRules = 'root' | 'cardSubheader' | 'nameHeader' | 'fileSizeHeader' | 'uploadIcon' | 'button' | 'centeredLabel' | 'cardHeaderContent' | 'cardHeaderContentTitle';

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
        paddingBottom: theme.spacing.unit,
        height: '100%'
    },
    cardSubheader: {
        paddingTop: 0,
        paddingBottom: 0,
        minHeight: 8 * theme.spacing.unit,
    },
    cardHeaderContent: {
        display: 'flex',
        paddingRight: 2 * theme.spacing.unit,
        justifyContent: 'space-between',
    },
    cardHeaderContentTitle: {
        paddingLeft: theme.spacing.unit,
        paddingTop: 2 * theme.spacing.unit,
        paddingRight: 2 * theme.spacing.unit,
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
        marginTop: '8px'
    },
    centeredLabel: {
        fontSize: '0.875rem',
        textAlign: 'center'
    },
});

export const CollectionPanelFilesComponent = ({ onItemMenuOpen, onSearchChange, onOptionsMenuOpen, onUploadDataClick, classes,
    isWritable, isLoading, tooManyFiles, loadFilesFunc, ...treeProps }: CollectionPanelFilesProps & WithStyles<CssRules>) => {
    const { useState, useEffect } = React;
    const [searchValue, setSearchValue] = useState('');

    useEffect(() => {
        onSearchChange(searchValue);
    }, [onSearchChange, searchValue]);

    return (<Card data-cy='collection-files-panel' className={classes.root}>
        <CardHeader
            title={
                <div className={classes.cardHeaderContent}>
                    <span className={classes.cardHeaderContentTitle}>Files</span>
                    <SearchInput
                        selfClearProp={''}
                        value={searchValue}
                        label='Search files'
                        onSearch={setSearchValue} />
                </div>
            }
            className={classes.cardSubheader}
            classes={{ action: classes.button }}
            action={<>
                {isWritable &&
                    <Button
                        data-cy='upload-button'
                        onClick={onUploadDataClick}
                        variant='contained'
                        color='primary'
                        size='small'>
                        <DownloadIcon className={classes.uploadIcon} />
                    Upload data
                </Button>}
                {!tooManyFiles &&
                    <Tooltip title="More options" disableFocusListener>
                        <IconButton
                            data-cy='collection-files-panel-options-btn'
                            onClick={(ev) => onOptionsMenuOpen(ev, isWritable)}>
                            <CustomizeTableIcon />
                        </IconButton>
                    </Tooltip>}
            </>
            } />
        {tooManyFiles
            ? <div className={classes.centeredLabel}>
                File listing may take some time, please click to browse: <Button onClick={loadFilesFunc}><DownloadIcon />Show files</Button>
            </div>
            : <>
                <Grid container justify="space-between">
                    <Typography variant="caption" className={classes.nameHeader}>
                        Name
                    </Typography>
                    <Typography variant="caption" className={classes.fileSizeHeader}>
                        File size
                    </Typography>
                </Grid>
                {isLoading
                    ? <div className={classes.centeredLabel}><CircularProgress /></div>
                    : <div style={{ height: 'calc(100% - 60px)' }}>
                        <FileTree
                            onMenuOpen={(ev, item) => onItemMenuOpen(ev, item, isWritable)}
                            {...treeProps} /></div>}
            </>
        }
    </Card>);
};

export const CollectionPanelFiles = withStyles(styles)(CollectionPanelFilesComponent);
