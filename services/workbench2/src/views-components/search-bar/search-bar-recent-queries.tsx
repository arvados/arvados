// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { List, ListItem, ListItemText } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { SearchBarSelectedItem } from "store/search-bar/search-bar-reducer";

type CssRules = 'root' | 'listItem' | 'listItemText';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        padding: '0px'
    },
    listItem: {
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(2),
    },
    listItemText: {
        fontSize: '0.8125rem',
        color: theme.palette.grey["900"]
    }
});

export interface SearchBarRecentQueriesDataProps {
    selectedItem: SearchBarSelectedItem;
    recentQueries: string[];
}

export interface SearchBarRecentQueriesActionProps {
    onSearch: (searchValue: string) => void;
}

type SearchBarRecentQueriesProps = SearchBarRecentQueriesDataProps & SearchBarRecentQueriesActionProps & WithStyles<CssRules>;

export const SearchBarRecentQueries = withStyles(styles)(
    ({ classes, onSearch, selectedItem, recentQueries }: SearchBarRecentQueriesProps) =>
        <List component="nav" className={classes.root}>
            {recentQueries.map((query, index) =>
                <ListItem button key={index} className={classes.listItem} selected={`RQ-${index}-${query}` === selectedItem.id}>
                    <ListItemText disableTypography
                        secondary={query}
                        onClick={() => onSearch(query)}
                        className={classes.listItemText} />
                </ListItem>
            )}
        </List>);
