// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid, IconButton, Badge, StyleRulesCallback, withStyles, WithStyles, Button, MenuItem } from "@material-ui/core";
import NotificationsIcon from "@material-ui/icons/Notifications";
import PersonIcon from "@material-ui/icons/Person";
import HelpIcon from "@material-ui/icons/Help";
import SearchBar from "./search-bar/search-bar";
import Breadcrumbs, { Breadcrumb } from "../breadcrumbs/breadcrumbs";
import DropdownMenu from "./dropdown-menu/dropdown-menu";
import { User } from "../../models/user";

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
    breadcrumbs: Breadcrumb[];
    user?: User;
    menuItems: MainAppBarMenuItems;
}

export interface MainAppBarActionProps {
    onSearch: (searchText: string) => void;
    onBreadcrumbClick: (breadcrumb: Breadcrumb) => void;
    onMenuItemClick: (menuItem: MainAppBarMenuItem) => void;
}

type MainAppBarProps = MainAppBarDataProps & MainAppBarActionProps & WithStyles<CssRules>;

export class MainAppBar extends React.Component<MainAppBarProps> {

    render() {
        const { classes, searchText, breadcrumbs, searchDebounce } = this.props;
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
                        {
                            this.props.user && <SearchBar
                                value={searchText}
                                onSearch={this.props.onSearch}
                                debounce={searchDebounce}
                            />
                        }
                    </Grid>
                    <Grid item xs={3} container alignItems="center" justify="flex-end">
                        {
                            this.props.user ? this.renderMenuForUser() : this.renderMenuForAnonymous()
                        }
                    </Grid>
                </Grid>
            </Toolbar>
            {
                this.props.user && <Toolbar>
                    <Breadcrumbs items={breadcrumbs} onClick={this.props.onBreadcrumbClick} />
                </Toolbar>
            }
        </AppBar>;
    }

    renderMenuForUser = () => {
        const { user } = this.props;
        return (
            <>
                <IconButton color="inherit">
                    <Badge badgeContent={3} color="primary">
                        <NotificationsIcon />
                    </Badge>
                </IconButton>
                <DropdownMenu icon={PersonIcon} id="account-menu">
                    <MenuItem>{this.getUserFullname()}</MenuItem>
                    {this.renderMenuItems(this.props.menuItems.accountMenu)}
                </DropdownMenu>
                <DropdownMenu icon={HelpIcon} id="help-menu">
                    {this.renderMenuItems(this.props.menuItems.helpMenu)}
                </DropdownMenu>
            </>
        );
    }

    renderMenuForAnonymous = () => {
        return this.props.menuItems.anonymousMenu.map((item, index) => (
            <Button key={index} color="inherit" onClick={() => this.props.onMenuItemClick(item)}>{item.label}</Button>
        ));
    }

    renderMenuItems = (menuItems: MainAppBarMenuItem[]) => {
        return menuItems.map((item, index) => (
            <MenuItem key={index} onClick={() => this.props.onMenuItemClick(item)}>{item.label}</MenuItem>
        ));
    }

    getUserFullname = () => {
        const { user } = this.props;
        return user ? `${user.firstName} ${user.lastName}` : "";
    }

}

type CssRules = "appBar";

const styles: StyleRulesCallback<CssRules> = theme => ({
    appBar: {
        backgroundColor: "#692498"
    }
});

export default withStyles(styles)(MainAppBar);