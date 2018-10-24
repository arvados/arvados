// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, ListItem, ListItemText } from '@material-ui/core';
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

export interface SearchBarAutocompleteViewActionProps {
    navigateTo: (uuid: string) => void;
}

type SearchBarAutocompleteViewProps = SearchBarAutocompleteViewDataProps & SearchBarAutocompleteViewActionProps & WithStyles<CssRules>;

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes, searchResults, searchValue, navigateTo }: SearchBarAutocompleteViewProps) =>
        <Paper className={classes.searchView}>
            {searchResults && <List component="nav" className={classes.list}>
                {searchResults.map((item: GroupContentsResource) =>
                    <ListItem button key={item.uuid}>
                        <ListItemText secondary={getFormattedText(item.name, searchValue)} onClick={() => navigateTo(item.uuid)} />
                    </ListItem>
                )}
            </List>}
        </Paper>
);

const getFormattedText = (textToHighlight: string, searchString = '') => {
    return <Highlighter searchWords={[searchString]} autoEscape={true} textToHighlight={textToHighlight} />;
};