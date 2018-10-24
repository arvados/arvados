// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { withStyles, WithStyles, StyleRulesCallback, List, ListItem, ListItemText } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';

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

export interface SearchBarRenderRecentQueriesActionProps {
    onSearch: (searchValue: string) => void;
    loadRecentQueries: () => string[];
}

type SearchBarRenderRecentQueriesProps = SearchBarRenderRecentQueriesActionProps & WithStyles<CssRules>;

export const SearchBarRenderRecentQueries = withStyles(styles)(
    ({ classes, onSearch, loadRecentQueries }: SearchBarRenderRecentQueriesProps) =>
        <List component="nav" className={classes.root}>
            {loadRecentQueries().map((query, index) =>
                <ListItem button key={index} className={classes.listItem}>
                    <ListItemText disableTypography 
                        secondary={query} 
                        onClick={() => onSearch(query)} 
                        className={classes.listItemText} />
                </ListItem>
            )}
        </List>);