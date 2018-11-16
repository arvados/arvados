// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { MenuItem } from "@material-ui/core";
import { User, getUserFullname } from "~/models/user";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { UserPanelIcon } from "~/components/icon/icon";
import { DispatchProp, connect } from 'react-redux';
import { logout } from "~/store/auth/auth-action";
import { RootState } from "~/store/store";
import { openCurrentTokenDialog } from '../../store/current-token-dialog/current-token-dialog-actions';
import { openVirtualMachines } from "~/store/virtual-machines/virtual-machines-actions";

interface AccountMenuProps {
    user?: User;
}

const mapStateToProps = (state: RootState): AccountMenuProps => ({
    user: state.auth.user
});

export const AccountMenu = connect(mapStateToProps)(
    ({ user, dispatch }: AccountMenuProps & DispatchProp<any>) =>
        user
            ? <DropdownMenu
                icon={<UserPanelIcon />}
                id="account-menu"
                title="Account Management">
                <MenuItem>
                    {getUserFullname(user)}
                </MenuItem>
                <MenuItem onClick={() => dispatch(openVirtualMachines())}>Virtual Machines</MenuItem>
                <MenuItem onClick={() => dispatch(openCurrentTokenDialog)}>Current token</MenuItem>
                <MenuItem>My account</MenuItem>
                <MenuItem onClick={() => dispatch(logout())}>Logout</MenuItem>
            </DropdownMenu>
            : null);
