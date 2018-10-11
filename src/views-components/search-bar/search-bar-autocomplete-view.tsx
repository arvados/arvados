// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List } from '@material-ui/core';
import { RecentQueriesItem } from '~/views-components/search-bar/search-bar-view';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import Highlighter from "react-highlight-words";

type CssRules = 'list' | 'searchView';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        list: {
            padding: 0
        },
        searchView: {
            borderRadius: `0 0 ${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px`
        }
    };
};

export interface SearchBarAutocompleteViewDataProps {
    searchResults?: GroupContentsResource[];
    searchValue?: string;
}

type SearchBarAutocompleteViewProps = SearchBarAutocompleteViewDataProps & WithStyles<CssRules>;

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes, searchResults, searchValue }: SearchBarAutocompleteViewProps) =>
        <Paper className={classes.searchView}>
            {searchResults && <List component="nav" className={classes.list}>
                {searchResults.map((item: GroupContentsResource) => {
                    return <RecentQueriesItem key={item.uuid} text={getFormattedText(item.name, searchValue)} />;
                })}
            </List>}
        </Paper>
);

const getFormattedText = (textToHighlight: string, searchString = '') => {
    return <Highlighter searchWords={[searchString]} autoEscape={true} textToHighlight={textToHighlight} />;
};