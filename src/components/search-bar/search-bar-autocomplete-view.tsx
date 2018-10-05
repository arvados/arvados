// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, Button } from '@material-ui/core';
import { renderRecentQueries } from '~/components/search-bar/search-bar';

type CssRules = 'list';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        list: {
            padding: '0px'
        }
    };
};

interface SearchBarAutocompleteViewProps {
}

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes }: SearchBarAutocompleteViewProps & WithStyles<CssRules>) =>
        <Paper>
            <List component="nav" className={classes.list}>
                {renderRecentQueries('AUTOCOMPLETE VIEW')}
            </List>
        </Paper>
);