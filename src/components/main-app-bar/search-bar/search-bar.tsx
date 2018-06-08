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
    onChange: (value: string) => any;
    onSubmit: () => any;
}

type SearchBarProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>

class SearchBar extends React.Component<SearchBarProps> {
    render() {
        const { classes } = this.props
        return <Paper className={classes.container}>
            <form onSubmit={this.handleSubmit}>
                <input
                    className={classes.input}
                    onChange={this.handleChange}
                    placeholder="Search"
                    value={this.props.value}
                />
                <IconButton className={classes.button}>
                    <SearchIcon />
                </IconButton>
            </form>
        </Paper>
    }

    handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        this.props.onSubmit();
    }

    handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        this.props.onChange(event.target.value);
    }

}

type CssRules = 'container' | 'input' | 'button'

const styles: StyleRulesCallback<CssRules> = theme => {
    const { unit } = theme.spacing
    return {
        container: {
            position: 'relative'
        },
        input: {
            border: 'none',
            padding: unit,
            paddingRight: unit * 4,
            borderRadius: unit / 4
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