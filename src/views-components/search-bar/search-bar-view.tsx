// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    IconButton,
    Paper,
    StyleRulesCallback,
    withStyles,
    WithStyles,
    Tooltip,
    InputAdornment, Input,
    ClickAwayListener
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import { ArvadosTheme } from '~/common/custom-theme';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import {
    SearchBarBasicView,
    SearchBarBasicViewDataProps,
    SearchBarBasicViewActionProps
} from '~/views-components/search-bar/search-bar-basic-view';
import {
    SearchBarAutocompleteView,
    SearchBarAutocompleteViewDataProps,
    SearchBarAutocompleteViewActionProps
} from '~/views-components/search-bar/search-bar-autocomplete-view';
import {
    SearchBarAdvancedView,
    SearchBarAdvancedViewDataProps,
    SearchBarAdvancedViewActionProps
} from '~/views-components/search-bar/search-bar-advanced-view';

type CssRules = 'container' | 'containerSearchViewOpened' | 'input' | 'view';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => {
    return {
        container: {
            position: 'relative',
            width: '100%',
            borderRadius: theme.spacing.unit / 2
        },
        containerSearchViewOpened: {
            position: 'relative',
            width: '100%',
            borderRadius: `${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px 0 0`
        },
        input: {
            border: 'none',
            padding: `0px ${theme.spacing.unit}px`
        },
        view: {
            position: 'absolute',
            width: '100%',
            zIndex: 1
        }
    };
};

export type SearchBarDataProps = SearchBarViewDataProps
    & SearchBarAutocompleteViewDataProps
    & SearchBarAdvancedViewDataProps
    & SearchBarBasicViewDataProps;

interface SearchBarViewDataProps {
    searchValue: string;
    currentView: string;
    isPopoverOpen: boolean;
    debounce?: number;
}

export type SearchBarActionProps = SearchBarViewActionProps
    & SearchBarAutocompleteViewActionProps
    & SearchBarAdvancedViewActionProps
    & SearchBarBasicViewActionProps;

interface SearchBarViewActionProps {
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
    onSetView: (currentView: string) => void;
    closeView: () => void;
    openSearchView: () => void;
    loadRecentQueries: () => string[];
}

type SearchBarViewProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

export const SearchBarView = withStyles(styles)(
    (props : SearchBarViewProps) => {
        const { classes, isPopoverOpen, closeView, searchValue, openSearchView, onChange, onSubmit } = props;
        return (
            <ClickAwayListener onClickAway={closeView}>
                <Paper className={isPopoverOpen ? classes.containerSearchViewOpened : classes.container} >
                    <form onSubmit={onSubmit}>
                        <Input
                            className={classes.input}
                            onChange={onChange}
                            placeholder="Search"
                            value={searchValue}
                            fullWidth={true}
                            disableUnderline={true}
                            onClick={openSearchView}
                            endAdornment={
                                <InputAdornment position="end">
                                    <Tooltip title='Search'>
                                        <IconButton>
                                            <SearchIcon />
                                        </IconButton>
                                    </Tooltip>
                                </InputAdornment>
                            } />
                    </form>
                    <div className={classes.view}>
                        {isPopoverOpen && getView({...props})}
                    </div>
                </Paper >
            </ClickAwayListener>
        );
    }
);

const getView = (props: SearchBarViewProps) => {
    const { onSetView, loadRecentQueries, savedQueries, deleteSavedQuery, searchValue,
        searchResults, saveQuery, onSearch, navigateTo, editSavedQuery, tags, currentView } = props;
    switch (currentView) {
        case SearchView.AUTOCOMPLETE:
            return <SearchBarAutocompleteView
                navigateTo={navigateTo}
                searchResults={searchResults}
                searchValue={searchValue} />;
        case SearchView.ADVANCED:
            return <SearchBarAdvancedView
                onSetView={onSetView}
                saveQuery={saveQuery}
                tags={tags} />;
        default:
            return <SearchBarBasicView
                onSetView={onSetView}
                onSearch={onSearch}
                loadRecentQueries={loadRecentQueries}
                savedQueries={savedQueries}
                deleteSavedQuery={deleteSavedQuery}
                editSavedQuery={editSavedQuery} />;
    }
};
