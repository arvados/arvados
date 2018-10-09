// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, ListItem, ListItemText } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { RecentQueriesItem } from '~/views-components/search-bar/search-bar-view';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import Highlighter from "react-highlight-words";

type CssRules = 'list';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    list: {
        padding: 0
    }
});

export interface SearchBarAutocompleteViewDataProps {
    searchResults?: GroupContentsResource[];
    searchValue?: string;
}

type SearchBarAutocompleteViewProps = SearchBarAutocompleteViewDataProps & WithStyles<CssRules>;

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes, searchResults, searchValue }: SearchBarAutocompleteViewProps ) =>
        <Paper>
            {searchResults &&  <List component="nav" className={classes.list}>
                {searchResults.map((item: GroupContentsResource) => {
                    return <RecentQueriesItem key={item.uuid} text={getFormattedText(item.name, searchValue)} />;
                })}
            </List>}
        </Paper>
);

const getFormattedText = (textToHighlight: string, searchString = '') => {
    return <Highlighter searchWords={[searchString]} autoEscape={true} textToHighlight={textToHighlight} />;
};