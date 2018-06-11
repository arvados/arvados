// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { AppBar, Toolbar, Typography, Grid, IconButton, Badge, Paper, Input, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import NotificationsIcon from '@material-ui/icons/Notifications';
import PersonIcon from '@material-ui/icons/Person';
import HelpIcon from '@material-ui/icons/Help';
import SearchIcon from '@material-ui/icons/Search';
import { AppBarProps } from '@material-ui/core/AppBar';

interface SearchBarDataProps {
    value: string;
}

interface SearchBarActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>

interface SearchBarState {
    value: string;
    prevValue: string;
}

const DEFAULT_SEARCH_DEBOUNCE = 1000;

class SearchBar extends React.Component<SearchBarProps> {

    state: SearchBarState = {
        value: "",
        prevValue: ""
    }

    timeout: NodeJS.Timer;

    render() {
        const { classes } = this.props
        return <Paper className={classes.container}>
            <form onSubmit={this.handleSubmit}>
                <input
                    className={classes.input}
                    onChange={this.handleChange}
                    placeholder="Search"
                    value={this.state.value}
                />
                <IconButton className={classes.button}>
                    <SearchIcon />
                </IconButton>
            </form>
        </Paper>
    }

    componentWillReceiveProps(nextProps: SearchBarProps) {
        if (nextProps.value !== this.props.value) {
            this.setState({ value: nextProps.value });
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
        this.timeout = setTimeout(
            () => this.props.onSearch(this.state.value),
            this.props.debounce || DEFAULT_SEARCH_DEBOUNCE
        );

    }

}

type CssRules = 'container' | 'input' | 'button'

const styles: StyleRulesCallback<CssRules> = theme => {
    const { unit } = theme.spacing
    return {
        container: {
            position: 'relative',
            width: '100%'
        },
        input: {
            border: 'none',
            borderRadius: unit / 4,
            boxSizing: 'border-box',
            padding: unit,
            paddingRight: unit * 4,
            width: '100%',
        },
        button: {
            position: 'absolute',
            top: unit / 2,
            right: unit / 2,
            width: unit * 3,
            height: unit * 3
        }
    }
}

export default withStyles(styles)(SearchBar)