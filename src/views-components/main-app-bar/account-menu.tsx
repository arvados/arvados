// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { MenuItem } from "@material-ui/core";
import { User, getUserFullname } from "~/models/user";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { UserPanelIcon } from "~/components/icon/icon";
import { DispatchProp, connect } from 'react-redux';
import { logout } from '~/store/auth/auth-action';
import { RootState } from "~/store/store";
import { openCurrentTokenDialog } from '~/store/current-token-dialog/current-token-dialog-actions';
import { openRepositoriesPanel } from "~/store/repositories/repositories-actions";
import {
    navigateToSiteManager,
    navigateToSshKeysUser,
    navigateToMyAccount
} from '~/store/navigation/navigation-action';
import { openUserVirtualMachines } from "~/store/virtual-machines/virtual-machines-actions";

interface AccountMenuProps {
    user?: User;
    currentRoute: string;
}

const mapStateToProps = (state: RootState): AccountMenuProps => ({
    user: state.auth.user,
    currentRoute: state.router.location ? state.router.location.pathname : ''
});

export const AccountMenu = connect(mapStateToProps)(
    ({ user, dispatch, currentRoute }: AccountMenuProps & DispatchProp<any>) =>
        user
            ? <DropdownMenu
                icon={<UserPanelIcon />}
                id="account-menu"
                title="Account Management"
                key={currentRoute}>
                <MenuItem>
                    {getUserFullname(user)}
                </MenuItem>
                <MenuItem onClick={() => dispatch(openUserVirtualMachines())}>Virtual Machines</MenuItem>
                {!user.isAdmin && <MenuItem onClick={() => dispatch(openRepositoriesPanel())}>Repositories</MenuItem>}
                <MenuItem onClick={() => dispatch(openCurrentTokenDialog)}>Current token</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToSshKeysUser)}>Ssh Keys</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToSiteManager)}>Site Manager</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToMyAccount)}>My account</MenuItem>
                <MenuItem onClick={() => dispatch(logout())}>Logout</MenuItem>
            </DropdownMenu>
            : null);
