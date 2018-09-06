// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Link } from "react-router-dom";
import { User } from "~/models/user";
import { SearchBar } from "~/components/search-bar/search-bar";
import { Routes } from '~/routes/routes';
import { NotificationsMenu } from "~/views-components/main-app-bar/notifications-menu";
import { AccountMenu } from "~/views-components/main-app-bar/account-menu";
import { AnonymousMenu } from "~/views-components/main-app-bar/anonymous-menu";

type CssRules = 'link';

const styles: StyleRulesCallback<CssRules> = () => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    }
});

interface MainAppBarDataProps {
    searchText: string;
    searchDebounce?: number;
    user?: User;
}

export interface MainAppBarActionProps {
    onSearch: (searchText: string) => void;
}

export type MainAppBarProps = MainAppBarDataProps & MainAppBarActionProps & WithStyles<CssRules>;

export const MainAppBar = withStyles(styles)(
    (props: MainAppBarProps) => {
        return <AppBar position="static">
            <Toolbar>
                <Grid container justify="space-between">
                    <Grid container item xs={3} alignItems="center">
                        <Typography variant="headline" color="inherit" noWrap>
                            <Link to={Routes.ROOT} className={props.classes.link}>
                                arvados workbench
                            </Link>
                        </Typography>
                    </Grid>
                    <Grid
                        item
                        xs={6}
                        container
                        alignItems="center">
                        {props.user && <SearchBar
                            value={props.searchText}
                            onSearch={props.onSearch}
                            debounce={props.searchDebounce}
                        />}
                    </Grid>
                    <Grid
                        item
                        xs={3}
                        container
                        alignItems="center"
                        justify="flex-end">
                        {props.user
                            ? <>
                                <NotificationsMenu />
                                <AccountMenu />
                            </>
                            : <AnonymousMenu />}
                    </Grid>
                </Grid>
            </Toolbar>
        </AppBar>;
    }
);
