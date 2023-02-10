// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { Badge, MenuItem } from "@material-ui/core";
import { DropdownMenu } from "components/dropdown-menu/dropdown-menu";
import { NotificationIcon } from "components/icon/icon";
import bannerActions from "store/banner/banner-action";
import { BANNER_LOCAL_STORAGE_KEY } from "views-components/baner/banner";
import { RootState } from "store/store";

const mapStateToProps = (state: RootState): NotificationsMenuProps => ({
    isOpen: state.banner.isOpen,
    bannerUUID: state.auth.config.clusterConfig.Workbench.BannerUUID,
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openBanner: () => dispatch<any>(bannerActions.openBanner()),
});

type NotificationsMenuProps = {
    isOpen: boolean;
    bannerUUID?: string;
}

type NotificationsMenuComponentProps = NotificationsMenuProps & {
    openBanner: any;
}

export const NotificationsMenuComponent = (props: NotificationsMenuComponentProps) => {
    const { isOpen, openBanner } = props;
    const result = localStorage.getItem(BANNER_LOCAL_STORAGE_KEY);
    const menuItems: any[] = [];

    if (!isOpen && result) {
        menuItems.push(<MenuItem><span onClick={openBanner}>Restore Banner</span></MenuItem>);
    }

    if (menuItems.length === 0) {
        menuItems.push(<MenuItem>You are up to date</MenuItem>);
    }

    return (<DropdownMenu
        icon={
            <Badge
                badgeContent={0}
                color="primary">
                <NotificationIcon />
            </Badge>}
        id="account-menu"
        title="Notifications">
        {
            menuItems.map(item => item)
        }
    </DropdownMenu>);
}

export const NotificationsMenu = connect(mapStateToProps, mapDispatchToProps)(NotificationsMenuComponent);
