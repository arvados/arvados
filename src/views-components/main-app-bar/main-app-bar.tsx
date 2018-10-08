// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Link } from "react-router-dom";
import { User } from "~/models/user";
import { SearchBar } from "~/views-components/search-bar/search-bar";
import { Routes } from '~/routes/routes';
import { NotificationsMenu } from "~/views-components/main-app-bar/notifications-menu";
import { AccountMenu } from "~/views-components/main-app-bar/account-menu";
import { HelpMenu } from './help-menu';
import { ReactNode } from "react";

type CssRules = 'toolbar' | 'link';

const styles: StyleRulesCallback<CssRules> = () => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    },
    toolbar: {
        height: '56px'
    }
});

interface MainAppBarDataProps {
    // searchText: string;
    // searchDebounce?: number;
    user?: User;
    buildInfo?: string;
    children?: ReactNode;
}

// export interface MainAppBarActionProps {
//     onSearch: (searchText: string) => void;
// }

export type MainAppBarProps = MainAppBarDataProps & WithStyles<CssRules>;

export const MainAppBar = withStyles(styles)(
    (props: MainAppBarProps) => {
        return <AppBar position="absolute">
            <Toolbar className={props.classes.toolbar}>
                <Grid container justify="space-between">
                    <Grid container item xs={3} direction="column" justify="center">
                        <Typography variant="title" color="inherit" noWrap>
                            <Link to={Routes.ROOT} className={props.classes.link}>
                                arvados workbench
                            </Link>
                        </Typography>
                        <Typography variant="caption" color="inherit">{props.buildInfo}</Typography>
                    </Grid>
                    <Grid
                        item
                        xs={6}
                        container
                        alignItems="center">
                        {/* {props.user && <SearchBar
                            value={props.searchText}
                            onSearch={props.onSearch}
                            debounce={props.searchDebounce}
                        />
                        } */}
                    </Grid>
                    <Grid
                        item
                        xs={3}
                        container
                        alignItems="center"
                        justify="flex-end"
                        wrap="nowrap">
                        {props.user
                            ? <>
                                <NotificationsMenu />
                                <AccountMenu />
                                <HelpMenu />
                            </>
                            : <HelpMenu />}
                    </Grid>
                </Grid>
            </Toolbar>
            {props.children}
        </AppBar>;
    }
);
