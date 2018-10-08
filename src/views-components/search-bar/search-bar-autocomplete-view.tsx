// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List } from '@material-ui/core';
import { RenderRecentQueries } from '~/views-components/search-bar/search-bar-view';
import { GroupContentsResource } from '~/services/groups-service/groups-service';

type CssRules = 'list';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        list: {
            padding: '0px'
        }
    };
};

export interface SearchBarAutocompleteViewDataProps {
    searchResults?: GroupContentsResource[];
}

type SearchBarAutocompleteViewProps = SearchBarAutocompleteViewDataProps & WithStyles<CssRules>;

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes, searchResults }: SearchBarAutocompleteViewProps ) =>
        <Paper>
            {searchResults &&  <List component="nav" className={classes.list}>
                {searchResults.map((item) => {
                    return <RenderRecentQueries key={item.uuid} text={item.name} />;
                })}
            </List>}
        </Paper>
);