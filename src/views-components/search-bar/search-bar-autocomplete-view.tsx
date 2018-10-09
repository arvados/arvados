// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List } from '@material-ui/core';
import { RenderRecentQueries } from '~/views-components/search-bar/search-bar-view';

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

interface SearchBarAutocompleteViewProps {
}

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes }: SearchBarAutocompleteViewProps & WithStyles<CssRules>) =>
        <Paper className={classes.searchView}>
            <List component="nav" className={classes.list}>
                <RenderRecentQueries text='AUTOCOMPLETE VIEW' />
            </List>
        </Paper>
);