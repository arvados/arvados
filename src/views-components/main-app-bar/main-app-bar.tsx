// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { AppBar, Toolbar, Typography, Grid, IconButton, Badge, Button, MenuItem } from "@material-ui/core";
import { User, getUserFullname } from "../../models/user";
import { SearchBar } from "../../components/search-bar/search-bar";
import { Breadcrumbs, Breadcrumb } from "../../components/breadcrumbs/breadcrumbs";
import { DropdownMenu } from "../../components/dropdown-menu/dropdown-menu";
import { DetailsIcon, NotificationIcon, UserPanelIcon, HelpIcon } from "../../components/icon/icon";

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
    onContextMenu: (event: React.MouseEvent<HTMLElement>, breadcrumb: Breadcrumb) => void;
    onDetailsPanelToggle: () => void;
}

type MainAppBarProps = MainAppBarDataProps & MainAppBarActionProps;

export const MainAppBar: React.SFC<MainAppBarProps> = (props) => {
    return <AppBar position="static">
        <Toolbar>
            <Grid container justify="space-between">
                <Grid item xs={3}>
                    <Typography variant="headline" color="inherit" noWrap>
                        Arvados
                    </Typography>
                    <Typography variant="body1" color="inherit" noWrap >
                        Workbench 2
                    </Typography>
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
                        props.user ? renderMenuForUser(props) : renderMenuForAnonymous(props)
                    }
                </Grid>
            </Grid>
        </Toolbar>
        <Toolbar >
            {
                props.user && <Breadcrumbs
                    items={props.breadcrumbs}
                    onClick={props.onBreadcrumbClick}
                    onContextMenu={props.onContextMenu} />
            }
            { props.user && <IconButton color="inherit" onClick={props.onDetailsPanelToggle}>
                    <DetailsIcon />
                </IconButton>
            }
        </Toolbar>
    </AppBar>;
};

const renderMenuForUser = ({ user, menuItems, onMenuItemClick }: MainAppBarProps) => {
    return (
        <>
            <IconButton color="inherit">
                <Badge badgeContent={3} color="primary">
                    <NotificationIcon />
                </Badge>
            </IconButton>
            <DropdownMenu icon={<UserPanelIcon />} id="account-menu">
                <MenuItem>
                    {getUserFullname(user)}
                </MenuItem>
                {renderMenuItems(menuItems.accountMenu, onMenuItemClick)}
            </DropdownMenu>
            <DropdownMenu icon={<HelpIcon />} id="help-menu">
                {renderMenuItems(menuItems.helpMenu, onMenuItemClick)}
            </DropdownMenu>
        </>
    );
};

const renderMenuForAnonymous = ({ onMenuItemClick, menuItems }: MainAppBarProps) => {
    return menuItems.anonymousMenu.map((item, index) => (
        <Button key={index} color="inherit" onClick={() => onMenuItemClick(item)}>
            {item.label}
        </Button>
    ));
};

const renderMenuItems = (menuItems: MainAppBarMenuItem[], onMenuItemClick: (menuItem: MainAppBarMenuItem) => void) => {
    return menuItems.map((item, index) => (
        <MenuItem key={index} onClick={() => onMenuItemClick(item)}>
            {item.label}
        </MenuItem>
    ));
};
