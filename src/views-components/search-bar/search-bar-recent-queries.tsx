// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { withStyles, WithStyles, StyleRulesCallback, List, ListItem, ListItemText } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { SearchBarSelectedItem } from "store/search-bar/search-bar-reducer";

type CssRules = 'root' | 'listItem' | 'listItemText';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        padding: '0px'
    },
    listItem: {
        paddingLeft: theme.spacing.unit,
        paddingRight: theme.spacing.unit * 2,
    },
    listItemText: {
        fontSize: '0.8125rem',
        color: theme.palette.grey["900"]
    }
});

export interface SearchBarRecentQueriesDataProps {
    selectedItem: SearchBarSelectedItem;
}

export interface SearchBarRecentQueriesActionProps {
    onSearch: (searchValue: string) => void;
    loadRecentQueries: () => string[];
}

type SearchBarRecentQueriesProps = SearchBarRecentQueriesDataProps & SearchBarRecentQueriesActionProps & WithStyles<CssRules>;

export const SearchBarRecentQueries = withStyles(styles)(
    ({ classes, onSearch, loadRecentQueries, selectedItem }: SearchBarRecentQueriesProps) =>
        <List component="nav" className={classes.root}>
            {loadRecentQueries().map((query, index) =>
                <ListItem button key={index} className={classes.listItem} selected={`RQ-${index}-${query}` === selectedItem.id}>
                    <ListItemText disableTypography
                        secondary={query}
                        onClick={() => onSearch(query)}
                        className={classes.listItemText} />
                </ListItem>
            )}
        </List>);
