// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import withStyles from '@mui/styles/withStyles';
import { WithStyles } from '@mui/styles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { ResourceName } from 'views-components/data-explorer/renderers';

type CssRules = 'root' | 'list' | 'item';

const styles: CustomStyleRulesCallback<CssRules> = () => ({
    root: {
        width: '100%',
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
    const selection = Object.keys(state.resources).slice(0, 5);
    const recents = selection.map(uuid => state.resources[uuid]);
    return {
        items: recents
    };
};

export const RecentlyVisitedSection = connect(mapStateToProps)(withStyles(styles)(({items, classes}: {items: any[]} & WithStyles<CssRules>) => {
    return (
        <div className={classes.root}>
            <div>Recently Visited</div>
            <ul className={classes.list}>
                {items.map(item => <RecentlyVisitedItem item={item} classes={classes} />)}
            </ul>
        </div>
    )
}));

type ItemProps = {
    item: { name: string, uuid: string, modifiedAt: string }
} & WithStyles<CssRules>;


const RecentlyVisitedItem = ({item, classes}: ItemProps) => {
    return (
        <div className={classes.item}>
            <span>
                <ResourceName uuid={item.uuid} />
            </span>
            <span>
                <span>{item.uuid}</span>
                <span style={{marginLeft: '2rem'}}>{new Date(item.modifiedAt).toLocaleString()}</span>
            </span>
        </div>
    );
}