// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles, List, ListItem, ListItemText } from '@material-ui/core';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import Highlighter from "react-highlight-words";
import { SearchBarSelectedItem } from "store/search-bar/search-bar-reducer";

type CssRules = 'searchView' | 'list' | 'listItem';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        searchView: {
            borderRadius: `0 0 ${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px`
        },
        list: {
            padding: 0
        },
        listItem: {
            paddingLeft: theme.spacing.unit,
            paddingRight: theme.spacing.unit * 2,
        }
    };
};

export interface SearchBarAutocompleteViewDataProps {
    searchResults: GroupContentsResource[];
    searchValue?: string;
    selectedItem: SearchBarSelectedItem;
}

export interface SearchBarAutocompleteViewActionProps {
    navigateTo: (uuid: string) => void;
}

type SearchBarAutocompleteViewProps = SearchBarAutocompleteViewDataProps & SearchBarAutocompleteViewActionProps & WithStyles<CssRules>;

export const SearchBarAutocompleteView = withStyles(styles)(
    ({ classes, searchResults, searchValue, navigateTo, selectedItem }: SearchBarAutocompleteViewProps) => {
        return <Paper className={classes.searchView}>
            <List component="nav" className={classes.list}>
                <ListItem button className={classes.listItem} selected={!selectedItem || searchValue === selectedItem.id}>
                    <ListItemText secondary={searchValue}/>
                </ListItem>
                {searchResults.map((item: GroupContentsResource) =>
                    <ListItem button key={item.uuid} className={classes.listItem} selected={item.uuid === selectedItem.id}>
                        <ListItemText secondary={getFormattedText(item.name, searchValue)}
                                      onClick={() => navigateTo(item.uuid)}/>
                    </ListItem>
                )}
            </List>
        </Paper>;
    });

const getFormattedText = (textToHighlight: string, searchString = '') => {
    return <Highlighter searchWords={[searchString]} autoEscape={true} textToHighlight={textToHighlight} />;
};
