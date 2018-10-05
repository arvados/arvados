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
    List, ListItem, ListItemText, ListItemSecondaryAction, Button
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import { RemoveIcon } from '~/components/icon/icon';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { goToView } from '~/store/structured-search/structured-search-actions';
import { SearchView } from '~/store/structured-search/structured-search-reducer';

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
            fontSize: '14px',
            cursor: 'pointer'
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
    currentView: string;
}

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
    onSetView: (currentView: string) => void;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

interface SearchBarState {
    value: string;
    isSearchViewOpen: boolean;
}

const mapStateToProps = ({ structuredSearch }: RootState) => {
    return {
        currentView: structuredSearch.currentView
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onSetView: (currentView: string) => {
        dispatch<any>(goToView(currentView));
    }
});

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchBar = connect(mapStateToProps, mapDispatchToProps)(withStyles(styles)(
    class extends React.Component<SearchBarProps> {
        state: SearchBarState = {
            value: "",
            isSearchViewOpen: false
        };

        timeout: number;

        render() {
            const { classes, currentView } = this.props;
            return <Paper className={classes.container} >
                <form onSubmit={this.handleSubmit} className={classes.searchBar}>
                    <Input
                        autoComplete={''}
                        className={classes.input}
                        onChange={this.handleChange}
                        placeholder="Search"
                        value={this.state.value}
                        fullWidth={true}
                        disableUnderline={true}
                        onClick={this.toggleSearchView}
                        endAdornment={
                            <InputAdornment position="end">
                                <Tooltip title='Search'>
                                    <IconButton>
                                        <SearchIcon />
                                    </IconButton>
                                </Tooltip>
                            </InputAdornment>
                        } />
                    {this.state.isSearchViewOpen && this.getView(currentView)}
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

        toggleSearchView = () =>
            this.setState({ isSearchViewOpen: !this.state.isSearchViewOpen })

        getView = (currentView: string) => {
            switch (currentView) {
                case SearchView.BASIC:
                    return <Paper className={this.props.classes.searchView}>
                        <div className={this.props.classes.searchQueryList}>Saved search queries</div>
                        <List component="nav" className={this.props.classes.list}>
                            {this.renderSavedQueries('Test')}
                            {this.renderSavedQueries('Demo')}
                        </List>
                        <div className={this.props.classes.searchQueryList}>Recent search queries</div>
                        <List component="nav" className={this.props.classes.list}>
                            {this.renderRecentQueries('cos')}
                            {this.renderRecentQueries('testtest')}
                        </List>
                        <div className={this.props.classes.advanced} onClick={() => this.props.onSetView(SearchView.ADVANCED)}>Advanced search</div>
                    </Paper>;
                case SearchView.ADVANCED:
                    return <Paper>
                        <List component="nav" className={this.props.classes.list}>
                            {this.renderRecentQueries('ADVANCED VIEW')}
                        </List>
                        <Button onClick={() => this.props.onSetView(SearchView.BASIC)}>Back</Button>
                    </Paper>;
                case SearchView.AUTOCOMPLETE:
                    return <Paper>
                        <List component="nav" className={this.props.classes.list}>
                            {this.renderRecentQueries('AUTOCOMPLETE VIEW')}
                        </List>
                    </Paper>;
                default:
                    return '';
            }
        }

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
            if (event.target.value.length > 0) {
                this.props.onSetView(SearchView.AUTOCOMPLETE);
            } else {
                this.props.onSetView(SearchView.BASIC);
            }
        }
    }
));
