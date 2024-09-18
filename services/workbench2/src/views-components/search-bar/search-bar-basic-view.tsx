// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Paper } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import {
    SearchBarRecentQueries,
    SearchBarRecentQueriesActionProps
} from 'views-components/search-bar/search-bar-recent-queries';
import {
    SearchBarSavedQueries,
    SearchBarSavedQueriesDataProps,
    SearchBarSavedQueriesActionProps
} from 'views-components/search-bar/search-bar-save-queries';

type CssRules = 'advanced' | 'label' | 'root';

const styles: CustomStyleRulesCallback<CssRules> = theme => {
    return {
        root: {
            color: theme.palette.common.black,
            borderRadius: `0 0 ${theme.spacing(0.5)} ${theme.spacing(0.5)}`
        },
        advanced: {
            display: 'flex',
            justifyContent: 'flex-end',
            padding: theme.spacing(1),
            fontSize: '0.875rem',
            cursor: 'pointer',
            color: theme.palette.primary.main
        },
        label: {
            fontSize: '0.775rem',
            padding: `${theme.spacing(1)} ${theme.spacing(1)} `,
            color: theme.palette.grey["900"],
            background: 'white',
            textAlign: 'right',
            fontWeight: 'bold'
        }
    };
};

export type SearchBarBasicViewDataProps = SearchBarSavedQueriesDataProps;

export type SearchBarBasicViewActionProps = {
    onSetView: (currentView: string) => void;
    onSearch: (searchValue: string) => void;
} & SearchBarRecentQueriesActionProps & SearchBarSavedQueriesActionProps;

type SearchBarBasicViewProps = SearchBarBasicViewDataProps & SearchBarBasicViewActionProps & WithStyles<CssRules>;

export const SearchBarBasicView = withStyles(styles)(
    ({ classes, onSetView, loadRecentQueries, deleteSavedQuery, savedQueries, onSearch, editSavedQuery, selectedItem }: SearchBarBasicViewProps) =>
        <Paper className={classes.root}>
            <div className={classes.label}>{"Recent queries"}</div>
            <SearchBarRecentQueries
                onSearch={onSearch}
                loadRecentQueries={loadRecentQueries}
                selectedItem={selectedItem} />
            <div className={classes.label}>{"Saved queries"}</div>
            <SearchBarSavedQueries
                onSearch={onSearch}
                savedQueries={savedQueries}
                editSavedQuery={editSavedQuery}
                deleteSavedQuery={deleteSavedQuery}
                selectedItem={selectedItem} />
        </Paper>
);
