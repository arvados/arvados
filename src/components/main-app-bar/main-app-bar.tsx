// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid, IconButton, Badge, Paper, Input, StyleRulesCallback, withStyles, WithStyles } from "@material-ui/core";
import NotificationsIcon from "@material-ui/icons/Notifications";
import PersonIcon from "@material-ui/icons/Person";
import HelpIcon from "@material-ui/icons/Help";
import SearchIcon from "@material-ui/icons/Search";
import { AppBarProps } from "@material-ui/core/AppBar";
import SearchBar from "./search-bar/search-bar";
import Breadcrumbs from "./breadcrumbs/breadcrumbs"

type CssRules = "appBar"

const styles: StyleRulesCallback<CssRules> = theme => ({
    appBar: {
        backgroundColor: "#692498"
    }
})

export class MainAppBar extends React.Component<WithStyles<CssRules>> {
    render() {
        const { classes } = this.props
        return <AppBar className={classes.appBar} position="static">
            <Toolbar>
                <Grid
                    container
                    justify="space-between"
                >
                    <Grid item xs={3}>
                        <Typography variant="title" color="inherit" noWrap style={{ flexGrow: 1 }}>
                            <span>Arvados</span><br /><span style={{ fontSize: 12 }}>Workbench 2</span>
                        </Typography>
                    </Grid>
                    <Grid item xs={6} container alignItems="center">
                        <SearchBar value="" onChange={console.log} onSubmit={() => console.log("submit")} />
                    </Grid>
                    <Grid item xs={3} container alignItems="center" justify="flex-end">
                        <IconButton color="inherit">
                            <Badge badgeContent={3} color="primary">
                                <NotificationsIcon />
                            </Badge>
                        </IconButton>
                        <IconButton color="inherit">
                            <PersonIcon />
                        </IconButton>
                        <IconButton color="inherit">
                            <HelpIcon />
                        </IconButton>
                    </Grid>
                </Grid>
            </Toolbar>
            <Toolbar>
                <Breadcrumbs items={["Projects", "Project 1"]} />
            </Toolbar>
        </AppBar>
    }

}

export default withStyles(styles)(MainAppBar)