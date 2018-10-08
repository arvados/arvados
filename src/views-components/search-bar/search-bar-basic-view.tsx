// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List } from '@material-ui/core';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { RenderRecentQueries, RenderSavedQueries } from '~/views-components/search-bar/search-bar-view';

type CssRules = 'advanced' | 'searchQueryList' | 'list' | 'searchView';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        advanced: {
            display: 'flex',
            justifyContent: 'flex-end',
            paddingRight: theme.spacing.unit * 2,
            paddingBottom: theme.spacing.unit,
            fontSize: '14px',
            cursor: 'pointer'
        },
        searchQueryList: {
            padding: `${theme.spacing.unit / 2}px ${theme.spacing.unit}px `,
            background: '#f2f2f2',
            fontSize: '14px'
        },
        list: {
            padding: '0px'
        },
        searchView: {
            color: theme.palette.common.black
        }
    };
};

interface SearchBarBasicViewProps {
    setView: (currentView: string) => void;
}

export const SearchBarBasicView = withStyles(styles)(
    ({ classes, setView }: SearchBarBasicViewProps & WithStyles<CssRules>) =>
        <Paper className={classes.searchView}>
            <div className={classes.searchQueryList}>Saved search queries</div>
            <List component="nav" className={classes.list}>
                <RenderSavedQueries text="Test" />
                <RenderSavedQueries text="Demo" />
            </List>
            <div className={classes.searchQueryList}>Recent search queries</div>
            <List component="nav" className={classes.list}>
                <RenderRecentQueries text="cos" />
                <RenderRecentQueries text="testtest" />
            </List>
            <div className={classes.advanced} onClick={() => setView(SearchView.ADVANCED)}>Advanced search</div>
        </Paper>
);