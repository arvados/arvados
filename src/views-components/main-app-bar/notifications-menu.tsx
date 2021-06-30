// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Badge, MenuItem } from '@material-ui/core';
import { DropdownMenu } from "components/dropdown-menu/dropdown-menu";
import { NotificationIcon } from 'components/icon/icon';

export const NotificationsMenu = 
    () =>
        <DropdownMenu
            icon={
                <Badge
                    badgeContent={0}
                    color="primary">
                    <NotificationIcon />
                </Badge>}
            id="account-menu"
            title="Notifications">
            <MenuItem>You are up to date</MenuItem>
        </DropdownMenu>;

