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
    currentView: string;
    isPopoverOpen: boolean;
    debounce?: number;
}

export type SearchBarActionProps = SearchBarViewActionProps
    & SearchBarAutocompleteViewActionProps
    & SearchBarAdvancedViewActionProps
    & SearchBarBasicViewActionProps;

interface SearchBarViewActionProps {
    onSearch: (value: string) => any;
    searchDataOnEnter: (value: string) => void;
    onSetView: (currentView: string) => void;
    closeView: () => void;
    openSearchView: () => void;
    saveRecentQuery: (query: string) => void;
    loadRecentQueries: () => string[];
}

type SearchBarViewProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
}

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchBarView = withStyles(styles)(
    class extends React.Component<SearchBarViewProps> {
        state: SearchBarState = {
            value: ""
        };

        timeout: number;

        render() {
            const { classes, currentView, openSearchView, closeView, isPopoverOpen } = this.props;
            return <ClickAwayListener onClickAway={closeView}>
                <Paper className={isPopoverOpen ? classes.containerSearchViewOpened : classes.container} >
                    <form onSubmit={this.handleSubmit}>
                        <Input
                            className={classes.input}
                            onChange={this.handleChange}
                            placeholder="Search"
                            value={this.state.value}
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
                        {isPopoverOpen && this.getView(currentView)}
                    </div>
                </Paper >
            </ClickAwayListener>;
        }

        componentDidMount() {
            this.setState({ value: this.props.searchValue });
        }

        componentWillReceiveProps(nextProps: SearchBarViewProps) {
            if (nextProps.searchValue !== this.props.searchValue) {
                this.setState({ value: nextProps.searchValue });
            }
        }

        componentWillUnmount() {
            clearTimeout(this.timeout);
        }

        getView = (currentView: string) => {
            const { onSetView, loadRecentQueries, savedQueries, deleteSavedQuery, searchValue, 
                searchResults, saveQuery, onSearch, navigateTo, editSavedQuery, tags } = this.props;
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
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            clearTimeout(this.timeout);
            this.props.saveRecentQuery(this.state.value);
            this.props.searchDataOnEnter(this.state.value);
            this.props.loadRecentQueries();
        }

        // ToDo: nie pokazywac autocomplete jezeli jestesmy w advance
        // currentView ze state.searchBar.currentView
        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            clearTimeout(this.timeout);
            this.setState({ value: event.target.value });
            this.timeout = window.setTimeout(
                () => this.props.onSearch(this.state.value),
                this.props.debounce || DEFAULT_SEARCH_DEBOUNCE
            );
            if (event.target.value.length > 0) {
                this.props.onSetView(SearchView.AUTOCOMPLETE);
            } else {
                this.props.onSetView(SearchView.BASIC);
            }
        }
    }
);
