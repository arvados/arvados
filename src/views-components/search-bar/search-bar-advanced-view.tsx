// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, Button } from '@material-ui/core';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { RenderRecentQueries } from '~/views-components/search-bar/search-bar-view';

type CssRules = 'list';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        list: {
            padding: '0px'
        }
    };
};

interface SearchBarAdvancedViewProps {
    setView: (currentView: string) => void;
}

export const SearchBarAdvancedView = withStyles(styles)(
    ({ classes, setView }: SearchBarAdvancedViewProps & WithStyles<CssRules>) =>
        <Paper>
            <List component="nav" className={classes.list}>
                <RenderRecentQueries text='ADVANCED VIEW' />
            </List>
            <Button onClick={() => setView(SearchView.BASIC)}>Back</Button>
        </Paper>
);