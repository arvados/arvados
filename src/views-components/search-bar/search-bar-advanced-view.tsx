// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, Button } from '@material-ui/core';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { RecentQueriesItem } from '~/views-components/search-bar/search-bar-view';

type CssRules = 'list' | 'searchView';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        list: {
            padding: '0px'
        },
        searchView: {
            borderRadius: `0 0 ${theme.spacing.unit / 4}px ${theme.spacing.unit / 4}px`
        }
    };
};

interface SearchBarAdvancedViewProps {
    setView: (currentView: string) => void;
}

export const SearchBarAdvancedView = withStyles(styles)(
    ({ classes, setView }: SearchBarAdvancedViewProps & WithStyles<CssRules>) =>
        <Paper className={classes.searchView}>
            <List component="nav" className={classes.list}>
                <RecentQueriesItem text='ADVANCED VIEW' />
            </List>
            <Button onClick={() => setView(SearchView.BASIC)}>Back</Button>
        </Paper>
);