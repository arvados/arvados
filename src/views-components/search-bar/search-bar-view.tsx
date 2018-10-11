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
import { RemoveIcon } from '~/components/icon/icon';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import { SearchBarBasicView } from '~/views-components/search-bar/search-bar-basic-view';
import { SearchBarAdvancedView } from '~/views-components/search-bar/search-bar-advanced-view';
import { SearchBarAutocompleteView, SearchBarAutocompleteViewDataProps } from '~/views-components/search-bar/search-bar-autocomplete-view';
import { ArvadosTheme } from '~/common/custom-theme';

type CssRules = 'container' | 'input' | 'searchBar' | 'view';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => {
    return {
        container: {
            position: 'relative',
            width: '100%',
            borderRadius: '0px'
        },
        input: {
            border: 'none',
            padding: `0px ${theme.spacing.unit}px`
        },
        searchBar: {
            height: '30px'
        },
        view: {
            position: 'absolute',
            width: '100%'
        }
    };
};

type SearchBarDataProps = {
    searchValue: string;
    currentView: string;
    open: boolean;
} & SearchBarAutocompleteViewDataProps;

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
    onSetView: (currentView: string) => void;
    openView: () => void;
    closeView: () => void;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
}

interface RenderQueriesProps {
    text: string | JSX.Element;
}

export const RecentQueriesItem = (props: RenderQueriesProps) => {
    return <ListItem button>
        <ListItemText secondary={props.text} />
    </ListItem>;
};


export const RenderSavedQueries = (props: RenderQueriesProps) => {
    return <ListItem button>
        <ListItemText secondary={props.text} />
        <ListItemSecondaryAction>
            <Tooltip title="Remove">
                <IconButton aria-label="Remove">
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
            const { classes, currentView, openView, closeView, open } = this.props;
            return <ClickAwayListener onClickAway={() => closeView()}>
                <Paper className={classes.container} >
                    <form onSubmit={this.handleSubmit} className={classes.searchBar}>
                        <Input
                            className={classes.input}
                            onChange={this.handleChange}
                            placeholder="Search"
                            value={this.state.value}
                            fullWidth={true}
                            disableUnderline={true}
                            onClick={() => openView()}
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
                        {open && this.getView(currentView)}
                    </div>
                </Paper>
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
            switch (currentView) {
                case SearchView.BASIC:
                    return <SearchBarBasicView setView={this.props.onSetView} />;
                case SearchView.ADVANCED:
                    return <SearchBarAdvancedView setView={this.props.onSetView} />;
                case SearchView.AUTOCOMPLETE:
                    return <SearchBarAutocompleteView 
                                searchResults={this.props.searchResults} 
                                searchValue={this.props.searchValue} />;
                default:
                    return <SearchBarBasicView setView={this.props.onSetView} />;
            }
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            clearTimeout(this.timeout);
            this.props.onSearch(this.state.value);
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
