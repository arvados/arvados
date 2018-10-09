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
import { SearchBarAutocompleteView } from '~/views-components/search-bar/search-bar-autocomplete-view';

type CssRules = 'container' | 'input' | 'searchBar';

const styles: StyleRulesCallback<CssRules> = theme => {
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
        }
    };
};

interface SearchBarDataProps {
    value: string;
    currentView: string;
    open: boolean;
}

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
    onSetView: (currentView: string) => void;
    openView: () => void;
    closeView: () => void;
    saveQuery: (query: string) => void;
    loadQueries: () => string[];
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
}

interface RenderQueriesProps {
    text: string;
}

export const RenderRecentQueries = (props: RenderQueriesProps) => {
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
                        {open && this.getView(currentView)}
                    </form>
                </Paper >
            </ClickAwayListener>;
        }

        componentDidMount() {
            this.setState({ value: this.props.value });
        }

        componentWillReceiveProps(nextProps: SearchBarProps) {
            if (nextProps.value !== this.props.value) {
                this.setState({ value: nextProps.value });
            }
        }

        componentWillUnmount() {
            clearTimeout(this.timeout);
        }

        getView = (currentView: string) => {
            switch (currentView) {
                case SearchView.BASIC:
                    return <SearchBarBasicView setView={this.props.onSetView} recentQueries={this.props.loadQueries}/>;
                case SearchView.ADVANCED:
                    return <SearchBarAdvancedView setView={this.props.onSetView} />;
                case SearchView.AUTOCOMPLETE:
                    return <SearchBarAutocompleteView />;
                default:
                    return <SearchBarBasicView setView={this.props.onSetView} recentQueries={this.props.loadQueries}/>;
            }
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
                event.preventDefault();
                clearTimeout(this.timeout);
                this.props.saveQuery(this.state.value);
                this.props.onSearch(this.state.value);
                this.props.loadQueries();
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
