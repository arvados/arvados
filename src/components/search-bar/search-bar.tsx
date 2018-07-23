// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconButton, Paper, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';

type CssRules = 'container' | 'input' | 'button';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        container: {
            position: 'relative',
            width: '100%'
        },
        input: {
            border: 'none',
            borderRadius: theme.spacing.unit / 4,
            boxSizing: 'border-box',
            padding: theme.spacing.unit,
            paddingRight: theme.spacing.unit * 4,
            width: '100%',
        },
        button: {
            position: 'absolute',
            top: theme.spacing.unit / 2,
            right: theme.spacing.unit / 2,
            width: theme.spacing.unit * 3,
            height: theme.spacing.unit * 3
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
}

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchBar = withStyles(styles)(
    class extends React.Component<SearchBarProps> {
        state: SearchBarState = {
            value: ""
        };

        timeout: number;

        render() {
            const {classes} = this.props;
            return <Paper className={classes.container}>
                <form onSubmit={this.handleSubmit}>
                    <input
                        className={classes.input}
                        onChange={this.handleChange}
                        placeholder="Search"
                        value={this.state.value}
                    />
                    <IconButton className={classes.button}>
                        <SearchIcon/>
                    </IconButton>
                </form>
            </Paper>;
        }

        componentDidMount() {
            this.setState({value: this.props.value});
        }

        componentWillReceiveProps(nextProps: SearchBarProps) {
            if (nextProps.value !== this.props.value) {
                this.setState({value: nextProps.value});
            }
        }

        componentWillUnmount() {
            clearTimeout(this.timeout);
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            clearTimeout(this.timeout);
            this.props.onSearch(this.state.value);
        }

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            clearTimeout(this.timeout);
            this.setState({value: event.target.value});
            this.timeout = window.setTimeout(
                () => this.props.onSearch(this.state.value),
                this.props.debounce || DEFAULT_SEARCH_DEBOUNCE
            );

        }
    }
);
