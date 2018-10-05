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
    List, ListItem, ListItemText, ListItemSecondaryAction
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import { RemoveIcon } from '~/components/icon/icon';

type CssRules = 'container' | 'input' | 'advanced' | 'searchQueryList' | 'list' | 'searchView' | 'searchBar';

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
        advanced: {
            display: 'flex',
            justifyContent: 'flex-end',
            paddingRight: theme.spacing.unit * 2,
            paddingBottom: theme.spacing.unit,
            fontSize: '14px'
        },
        searchQueryList: {
            padding: `${theme.spacing.unit / 2}px ${theme.spacing.unit}px `,
            background: '#f2f2f2',
            fontSize: '14px'
        },
        list: {
            padding: '0px'
        },
        searchView: {
            color: theme.palette.common.black
        },
        searchBar: {
            height: '30px'
        }
    };
};

interface SearchBarDataProps {
    value: string;
}

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
    isSearchViewOpen: boolean;
}

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchBar = withStyles(styles)(
    class extends React.Component<SearchBarProps> {
        state: SearchBarState = {
            value: "",
            isSearchViewOpen: false
        };

        timeout: number;

        render() {
            const { classes } = this.props;
            return <Paper className={classes.container} onBlur={this.closeSearchView}>
                <form onSubmit={this.handleSubmit} className={classes.searchBar}>
                    <Input
                        autoComplete={''}
                        className={classes.input}
                        onChange={this.handleChange}
                        placeholder="Search"
                        value={this.state.value}
                        fullWidth={true}
                        disableUnderline={true}
                        onFocus={this.openSearchView}
                        endAdornment={
                            <InputAdornment position="end">
                                <Tooltip title='Search'>
                                    <IconButton>
                                        <SearchIcon />
                                    </IconButton>
                                </Tooltip>
                            </InputAdornment>
                        } />
                    {this.state.isSearchViewOpen && <Paper className={classes.searchView}>
                        <div className={classes.searchQueryList}>Saved search queries</div>
                        <List component="nav" className={classes.list}>
                            {this.renderSavedQueries('Test')}
                            {this.renderSavedQueries('Demo')}
                        </List>
                        <div className={classes.searchQueryList}>Recent search queries</div>
                        <List component="nav" className={classes.list}>
                            {this.renderRecentQueries('cos')}
                            {this.renderRecentQueries('testtest')}
                        </List>
                        <div className={classes.advanced}>Advanced search</div>
                    </Paper>}
                </form>
            </Paper>;
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

        closeSearchView = () =>
            this.setState({ isSearchViewOpen: false })


        openSearchView = () =>
            this.setState({ isSearchViewOpen: true })


        renderRecentQueries = (text: string) =>
            <ListItem button>
                <ListItemText secondary={text} />
            </ListItem>

        renderSavedQueries = (text: string) =>
            <ListItem button>
                <ListItemText secondary={text} />
                <ListItemSecondaryAction>
                    <Tooltip title="Remove">
                        <IconButton aria-label="Remove">
                            <RemoveIcon />
                        </IconButton>
                    </Tooltip>
                </ListItemSecondaryAction>
            </ListItem>

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

        }
    }
);
