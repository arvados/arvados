// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { MenuItem } from "@material-ui/core";
import { User } from "~/models/user";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { AdminMenuIcon } from "~/components/icon/icon";
import { DispatchProp, connect } from 'react-redux';
import { logout } from '~/store/auth/auth-action';
import { RootState } from "~/store/store";
import { openRepositoriesPanel } from "~/store/repositories/repositories-actions";
import { 
    navigateToSshKeysAdmin, navigateToKeepServices, navigateToComputeNodes,
    navigateToApiClientAuthorizations
} from '~/store/navigation/navigation-action';
import { openAdminVirtualMachines } from "~/store/virtual-machines/virtual-machines-actions";
import { navigateToUsers } from '~/store/navigation/navigation-action';

interface AdminMenuProps {
    user?: User;
}

const mapStateToProps = (state: RootState): AdminMenuProps => ({
    user: state.auth.user
});

export const AdminMenu = connect(mapStateToProps)(
    ({ user, dispatch }: AdminMenuProps & DispatchProp<any>) =>
        user
            ? <DropdownMenu
                icon={<AdminMenuIcon />}
                id="admin-menu"
                title="Admin Panel">
                <MenuItem onClick={() => dispatch(openRepositoriesPanel())}>Repositories</MenuItem>
                <MenuItem onClick={() => dispatch(openAdminVirtualMachines())}>Virtual Machines</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToSshKeysAdmin)}>Ssh Keys</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToApiClientAuthorizations)}>Api Tokens</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToUsers)}>Users</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToComputeNodes)}>Compute Nodes</MenuItem>
                <MenuItem onClick={() => dispatch(navigateToKeepServices)}>Keep Services</MenuItem>
                <MenuItem onClick={() => dispatch(logout())}>Logout</MenuItem>
            </DropdownMenu>
            : null);
