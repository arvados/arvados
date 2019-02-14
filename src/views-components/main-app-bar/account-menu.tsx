// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { MenuItem, Divider } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { User, getUserFullname } from "~/models/user";
import { DropdownMenu } from "~/components/dropdown-menu/dropdown-menu";
import { Link } from "react-router-dom";
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
    workbenchURL: string;
}

const mapStateToProps = (state: RootState): AccountMenuProps => ({
    user: state.auth.user,
    currentRoute: state.router.location ? state.router.location.pathname : '',
    workbenchURL: state.config.config.workbenchUrl,
});

const wb1URL = (route: string) => {
    const r = route.replace(/^\//, "");
    if (r.match(/^(projects|collections)\//)) {
        return r;
    } else if (r.match(/^processes\//)) {
        return r.replace(/^processes/, "container_requests");
    }
    return "";
};

type CssRules = 'link';

const styles: StyleRulesCallback<CssRules> = () => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    }
});

export const AccountMenu = withStyles(styles)(
    connect(mapStateToProps)(
        ({ user, dispatch, currentRoute, workbenchURL, classes }: AccountMenuProps & DispatchProp<any> & WithStyles<CssRules>) =>
            user
                ? <DropdownMenu
                    icon={<UserPanelIcon />}
                    id="account-menu"
                    title="Account Management"
                    key={currentRoute}>
                    <MenuItem disabled>
                        {getUserFullname(user)}
                    </MenuItem>
                    <MenuItem onClick={() => dispatch(openUserVirtualMachines())}>Virtual Machines</MenuItem>
                    {!user.isAdmin && <MenuItem onClick={() => dispatch(openRepositoriesPanel())}>Repositories</MenuItem>}
                    <MenuItem onClick={() => dispatch(openCurrentTokenDialog)}>Current token</MenuItem>
                    <MenuItem onClick={() => dispatch(navigateToSshKeysUser)}>Ssh Keys</MenuItem>
                    <MenuItem onClick={() => dispatch(navigateToSiteManager)}>Site Manager</MenuItem>
                    <MenuItem onClick={() => dispatch(navigateToMyAccount)}>My account</MenuItem>
                    <MenuItem>
                        <a href={`${workbenchURL.replace(/\/$/, "")}/${wb1URL(currentRoute)}`}
                            className={classes.link}>
                            Switch to Workbench v1</a></MenuItem>
                    <Divider />
                    <MenuItem onClick={() => dispatch(logout())}>Logout</MenuItem>
                </DropdownMenu>
                : null));
