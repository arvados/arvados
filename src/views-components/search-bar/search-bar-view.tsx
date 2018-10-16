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
    ListItem, ListItemText, ListItemSecondaryAction,
    ClickAwayListener
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import { RemoveIcon, EditSavedQueryIcon } from '~/components/icon/icon';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { SearchBarBasicView } from '~/views-components/search-bar/search-bar-basic-view';
import { SearchBarAdvancedView } from '~/views-components/search-bar/search-bar-advanced-view';
import { SearchBarAutocompleteView, SearchBarAutocompleteViewDataProps } from '~/views-components/search-bar/search-bar-autocomplete-view';
import { ArvadosTheme } from '~/common/custom-theme';
import { SearchBarAdvanceFormData } from '~/models/search-bar';

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

type SearchBarDataProps = {
    searchValue: string;
    currentView: string;
    isPopoverOpen: boolean;
    savedQueries: SearchBarAdvanceFormData[];
} & SearchBarAutocompleteViewDataProps;

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
    onSetView: (currentView: string) => void;
    closeView: () => void;
    saveRecentQuery: (query: string) => void;
    loadRecentQueries: () => string[];
    saveQuery: (data: SearchBarAdvanceFormData) => void;
    deleteSavedQuery: (id: number) => void;
    openSearchView: () => void;
    navigateTo: (uuid: string) => void;
    editSavedQuery: (data: SearchBarAdvanceFormData, id: number) => void;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
}

interface RenderRecentQueriesProps {
    text: string | JSX.Element;
    onSearch: (searchValue: string | JSX.Element) => void;
}

export const RenderRecentQueries = (props: RenderRecentQueriesProps) => {
    return <ListItem button>
        <ListItemText secondary={props.text} onClick={() => props.onSearch(props.text)} />
    </ListItem>;
};

interface RenderAutocompleteItemsProps {
    text: string | JSX.Element;
    navigateTo: (uuid: string) => void;
    uuid: string;
}

export const RenderAutocompleteItems = (props: RenderAutocompleteItemsProps) => {
    return <ListItem button>
        <ListItemText secondary={props.text} onClick={() => props.navigateTo(props.uuid)} />
    </ListItem>;
};

interface RenderSavedQueriesProps {
    text: string | JSX.Element;
    id: number;
    deleteSavedQuery: (id: number) => void;
    onSearch: (searchValue: string | JSX.Element) => void;
    editSavedQuery: (data: SearchBarAdvanceFormData, id: number) => void;
    data: SearchBarAdvanceFormData;
}

export const RenderSavedQueries = (props: RenderSavedQueriesProps) => {
    return <ListItem button>
        <ListItemText secondary={props.text} onClick={() => props.onSearch(props.text)} />
        <ListItemSecondaryAction>
            <Tooltip title="Edit">
                <IconButton aria-label="Edit" onClick={() => props.editSavedQuery(props.data, props.id)}>
                    <EditSavedQueryIcon />
                </IconButton>
            </Tooltip>
            <Tooltip title="Remove">
                <IconButton aria-label="Remove" onClick={() => props.deleteSavedQuery(props.id)}>
                    <RemoveIcon />
                </IconButton>
            </Tooltip>
        </ListItemSecondaryAction>
    </ListItem>;
};

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchBarView = withStyles(styles)(
    class extends React.Component<SearchBarProps> {
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

        componentWillReceiveProps(nextProps: SearchBarProps) {
            if (nextProps.searchValue !== this.props.searchValue) {
                this.setState({ value: nextProps.searchValue });
            }
        }

        componentWillUnmount() {
            clearTimeout(this.timeout);
        }

        getView = (currentView: string) => {
            const { onSetView, loadRecentQueries, savedQueries, deleteSavedQuery, searchValue, searchResults, saveQuery, onSearch, navigateTo, editSavedQuery } = this.props;
            switch (currentView) {
                case SearchView.BASIC:
                    return <SearchBarBasicView setView={onSetView} recentQueries={loadRecentQueries} savedQueries={savedQueries} deleteSavedQuery={deleteSavedQuery} onSearch={onSearch} editSavedQuery={editSavedQuery} />;
                case SearchView.ADVANCED:
                    return <SearchBarAdvancedView setView={onSetView} saveQuery={saveQuery} />;
                case SearchView.AUTOCOMPLETE:
                    return <SearchBarAutocompleteView
                        navigateTo={navigateTo}
                        searchResults={searchResults}
                        searchValue={searchValue} />;
                default:
                    return <SearchBarBasicView setView={onSetView} recentQueries={loadRecentQueries} savedQueries={savedQueries} deleteSavedQuery={deleteSavedQuery} onSearch={onSearch} editSavedQuery={editSavedQuery} />;
            }
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            clearTimeout(this.timeout);
            this.props.saveRecentQuery(this.state.value);
            this.props.onSearch(this.state.value);
            this.props.loadRecentQueries();
        }

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
