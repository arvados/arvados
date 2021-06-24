// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { withStyles, WithStyles, StyleRulesCallback, List, ListItem, ListItemText, ListItemSecondaryAction, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RemoveIcon, EditSavedQueryIcon } from 'components/icon/icon';
import { SearchBarAdvancedFormData } from 'models/search-bar';
import { SearchBarSelectedItem } from "store/search-bar/search-bar-reducer";
import { getQueryFromAdvancedData } from "store/search-bar/search-bar-actions";

type CssRules = 'root' | 'listItem' | 'listItemText' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        padding: '0px'
    },
    listItem: {
        paddingLeft: theme.spacing.unit,
        paddingRight: theme.spacing.unit * 2
    },
    listItemText: {
        fontSize: '0.8125rem',
        color: theme.palette.grey["900"]
    },
    button: {
        padding: '6px',
        marginRight: theme.spacing.unit
    }
});

export interface SearchBarSavedQueriesDataProps {
    savedQueries: SearchBarAdvancedFormData[];
    selectedItem: SearchBarSelectedItem;
}

export interface SearchBarSavedQueriesActionProps {
    onSearch: (searchValue: string) => void;
    deleteSavedQuery: (id: number) => void;
    editSavedQuery: (data: SearchBarAdvancedFormData, id: number) => void;
}

type SearchBarSavedQueriesProps = SearchBarSavedQueriesDataProps
    & SearchBarSavedQueriesActionProps
    & WithStyles<CssRules>;

export const SearchBarSavedQueries = withStyles(styles)(
    ({ classes, savedQueries, onSearch, editSavedQuery, deleteSavedQuery, selectedItem }: SearchBarSavedQueriesProps) =>
        <List component="nav" className={classes.root}>
            {savedQueries.map((query, index) =>
                <ListItem button key={index} className={classes.listItem} selected={`SQ-${index}-${query.queryName}` === selectedItem.id}>
                    <ListItemText disableTypography
                        secondary={query.queryName}
                        onClick={() => onSearch(getQueryFromAdvancedData(query))}
                        className={classes.listItemText} />
                    <ListItemSecondaryAction>
                        <Tooltip title="Edit">
                            <IconButton aria-label="Edit" onClick={() => editSavedQuery(query, index)} className={classes.button}>
                                <EditSavedQueryIcon />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title="Remove">
                            <IconButton aria-label="Remove" onClick={() => deleteSavedQuery(index)} className={classes.button}>
                                <RemoveIcon />
                            </IconButton>
                        </Tooltip>
                    </ListItemSecondaryAction>
                </ListItem>
            )}
    </List>);
