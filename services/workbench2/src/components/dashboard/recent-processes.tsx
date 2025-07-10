// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { Collapse } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { loadAllProcessesPanel } from 'store/all-processes-panel/all-processes-panel-action';
import { ProcessStatus } from 'views-components/data-explorer/renderers';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'root' | 'title' | 'list' | 'item';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    title: {
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        borderRadius: '4px',
        marginLeft: '1rem',
        padding: '4px',
        '&:hover': {
            background: 'lightgray',
        },
    },
    list: {
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'flex-start',
        width: '100%',
        marginLeft: '-1rem',
    },
    item: {
        padding: '8px',
        margin: '4px 0',
        width: '100%',
        background: '#fafafa',
        borderRadius: '8px',
        // Additional styles for better appearance
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        '&:hover': {
            background: 'lightgray',
        },
    },
});

const mapStateToProps = (state: RootState) => {
    const selection = (state.dataExplorer.allProcessesPanel?.items || []).slice(0, 3);
    const recents = selection.map(uuid => state.resources[uuid]);
    return {
        items: recents
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    loadAllProcessesPanel: () => dispatch<any>(loadAllProcessesPanel()),
});

type RecentProcessesProps = {
    items: any[];
    loadAllProcessesPanel: () => void;
} & WithStyles<CssRules>;

export const RecentProcessesSection = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(({items, loadAllProcessesPanel, classes}: RecentProcessesProps) => {
    useEffect(() => {
        loadAllProcessesPanel();
    }, [loadAllProcessesPanel]);

    const [isOpen, setIsOpen] = useState(true);

    return (
        <div className={classes.root}>
            <span className={classes.title} onClick={() => setIsOpen(!isOpen)}>Recent Processes</span>
            {isOpen ? <Collapse in={isOpen}>
                <ul className={classes.list}>
                    {items.map(item => <RecentProcessItem item={item} classes={classes} />)}
                </ul>
            </Collapse> : <div style={{margin: '1rem'}}><hr/></div>}
        </div>
    )
}));

type ItemProps = {
    item: { name: string, uuid: string, modifiedAt: string }
} & WithStyles<CssRules>;


const RecentProcessItem = ({item, classes}: ItemProps) => {
    return (
        <div className={classes.item}>
            <span>
                <ResourceName uuid={item.uuid} />
            </span>
            <span style={{display: 'flex'}}>
                <span><ProcessStatus uuid={item.uuid} /></span>
                <div style={{marginLeft: '2rem', width: '12rem'}}>{new Date(item.modifiedAt).toLocaleString()}</div>
            </span>
        </div>
    );
}