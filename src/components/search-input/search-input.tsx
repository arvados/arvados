// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconButton, StyleRulesCallback, withStyles, WithStyles, FormControl, InputLabel, Input, InputAdornment } from '@material-ui/core';
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

interface SearchInputDataProps {
    value: string;
}

interface SearchInputActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchInputProps = SearchInputDataProps & SearchInputActionProps & WithStyles<CssRules>;

interface SearchInputState {
    value: string;
}

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchInput = withStyles(styles)(
    class extends React.Component<SearchInputProps> {
        state: SearchInputState = {
            value: ""
        };

        timeout: number;

        render() {
            const { classes } = this.props;
            return <form onSubmit={this.handleSubmit}>
                <FormControl>
                    <InputLabel>Search</InputLabel>
                    <Input
                        type="text"
                        value={this.state.value}
                        onChange={this.handleChange}
                        endAdornment={
                            <InputAdornment position="end">
                                <IconButton
                                    onClick={this.handleSubmit}>
                                    <SearchIcon/>
                                </IconButton>
                            </InputAdornment>
                        }/>
                </FormControl>
            </form>;
        }

        componentDidMount() {
            this.setState({ value: this.props.value });
        }

        componentWillReceiveProps(nextProps: SearchInputProps) {
            if (nextProps.value !== this.props.value) {
                this.setState({ value: nextProps.value });
            }
        }

        componentWillUnmount() {
            clearTimeout(this.timeout);
        }

        handleSubmit = (event: React.FormEvent<HTMLElement>) => {
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
