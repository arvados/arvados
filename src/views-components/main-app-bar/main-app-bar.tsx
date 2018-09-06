// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid, IconButton, Badge, Button, MenuItem, Tooltip } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { Link } from "react-router-dom";
import { User, getUserFullname } from "~/models/user";
import { SearchBar } from "~/components/search-bar/search-bar";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { DetailsIcon, NotificationIcon, UserPanelIcon, HelpIcon } from "~/components/icon/icon";
import { Routes } from '~/routes/routes';
import { NotificationsMenu } from "~/views-components/main-app-bar/notifications-menu";
import { AccountMenu } from "~/views-components/main-app-bar/account-menu";
import { AnonymousMenu } from "~/views-components/main-app-bar/anonymous-menu";

type CssRules = 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    }
});

export interface MainAppBarMenuItem {
    label: string;
}

export interface MainAppBarMenuItems {
    accountMenu: MainAppBarMenuItem[];
    helpMenu: MainAppBarMenuItem[];
    anonymousMenu: MainAppBarMenuItem[];
}

interface MainAppBarDataProps {
    searchText: string;
    searchDebounce?: number;
    breadcrumbs: React.ComponentType<any>;
    user?: User;
    menuItems: MainAppBarMenuItems;
    buildInfo: string;
}

export interface MainAppBarActionProps {
    onSearch: (searchText: string) => void;
    onMenuItemClick: (menuItem: MainAppBarMenuItem) => void;
    onDetailsPanelToggle: () => void;
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
                        {/* <Typography variant="body1" color="inherit" noWrap >
                            {props.buildInfo}
                        </Typography> */}
                    </Grid>
                    <Grid item xs={6} container alignItems="center">
                        {
                            props.user && <SearchBar
                                value={props.searchText}
                                onSearch={props.onSearch}
                                debounce={props.searchDebounce}
                            />
                        }
                    </Grid>
                    <Grid item xs={3} container alignItems="center" justify="flex-end">
                        {
                            props.user
                                ? <>
                                    <NotificationsMenu />
                                    <AccountMenu />
                                </>
                                : <AnonymousMenu />
                        }
                    </Grid>
                </Grid>
            </Toolbar>
            {/* <Toolbar >
                {props.user && <props.breadcrumbs />}
                {props.user && <IconButton color="inherit" onClick={props.onDetailsPanelToggle}>
                    <Tooltip title="Additional Info">
                        <DetailsIcon />
                    </Tooltip>
                </IconButton>}
            </Toolbar> */}
        </AppBar>;
    }
);
