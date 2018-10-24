// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Paper, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import {
    SearchBarRenderRecentQueries,
    SearchBarRenderRecentQueriesActionProps 
} from '~/views-components/search-bar/search-bar-render-recent-queries';
import {
    SearchBarRenderSavedQueries,
    SearchBarRenderSavedQueriesDataProps,
    SearchBarRenderSavedQueriesActionProps
} from '~/views-components/search-bar/search-bar-render-save-queries';

type CssRules = 'advanced' | 'label' | 'root';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        root: {
            color: theme.palette.common.black,
            borderRadius: `0 0 ${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px`
        },
        advanced: {
            display: 'flex',
            justifyContent: 'flex-end',
            padding: theme.spacing.unit,
            fontSize: '0.875rem',
            cursor: 'pointer',
            color: theme.palette.primary.main
        },
        label: {
            fontSize: '0.875rem',
            padding: `${theme.spacing.unit / 2}px ${theme.spacing.unit}px `,
            color: theme.palette.grey["900"],
            background: theme.palette.grey["200"]
        }
    };
};

export type SearchBarBasicViewDataProps = SearchBarRenderSavedQueriesDataProps;

export type SearchBarBasicViewActionProps = {
    onSetView: (currentView: string) => void;
    onSearch: (searchValue: string) => void;
} & SearchBarRenderRecentQueriesActionProps & SearchBarRenderSavedQueriesActionProps;

type SearchBarBasicViewProps = SearchBarBasicViewDataProps & SearchBarBasicViewActionProps & WithStyles<CssRules>;

export const SearchBarBasicView = withStyles(styles)(
    ({ classes, onSetView, loadRecentQueries, deleteSavedQuery, savedQueries, onSearch, editSavedQuery }: SearchBarBasicViewProps) =>
        <Paper className={classes.root}>
            <div className={classes.label}>Recent search queries</div>
            <SearchBarRenderRecentQueries
                onSearch={onSearch}
                loadRecentQueries={loadRecentQueries} />
            <div className={classes.label}>Saved search queries</div>
            <SearchBarRenderSavedQueries
                onSearch={onSearch}
                savedQueries={savedQueries}
                editSavedQuery={editSavedQuery}
                deleteSavedQuery={deleteSavedQuery} />
            <div className={classes.advanced} onClick={() => onSetView(SearchView.ADVANCED)}>Advanced search</div>
        </Paper>
);