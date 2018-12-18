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
import * as NavigationAction from '~/store/navigation/navigation-action';
import { openAdminVirtualMachines } from "~/store/virtual-machines/virtual-machines-actions";
import { openUserPanel } from "~/store/users/users-actions";

interface AdminMenuProps {
    user?: User;
    currentRoute: string;
}

const mapStateToProps = (state: RootState): AdminMenuProps => ({
    user: state.auth.user,
    currentRoute: state.router.location ? state.router.location.pathname : ''
});

export const AdminMenu = connect(mapStateToProps)(
    ({ user, dispatch, currentRoute }: AdminMenuProps & DispatchProp<any>) =>
        user
            ? <DropdownMenu
                icon={<AdminMenuIcon />}
                id="admin-menu"
                title="Admin Panel"
                key={currentRoute}>
                <MenuItem onClick={() => dispatch(openRepositoriesPanel())}>Repositories</MenuItem>
                <MenuItem onClick={() => dispatch(openAdminVirtualMachines())}>Virtual Machines</MenuItem>
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToSshKeysAdmin)}>Ssh Keys</MenuItem>
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToApiClientAuthorizations)}>Api Tokens</MenuItem>
                <MenuItem onClick={() => dispatch(openUserPanel())}>Users</MenuItem>
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToGroups)}>Groups</MenuItem>}
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToComputeNodes)}>Compute Nodes</MenuItem>
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToKeepServices)}>Keep Services</MenuItem>
                <MenuItem onClick={() => dispatch(NavigationAction.navigateToLinks)}>Links</MenuItem>
            </DropdownMenu>
            : null);
