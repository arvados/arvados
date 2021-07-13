// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { MenuItem, Divider } from "@material-ui/core";
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { User, getUserDisplayName } from "models/user";
import { DropdownMenu } from "components/dropdown-menu/dropdown-menu";
import { UserPanelIcon } from "components/icon/icon";
import { DispatchProp, connect } from 'react-redux';
import { authActions, getNewExtraToken } from 'store/auth/auth-action';
import { RootState } from "store/store";
import { openTokenDialog } from 'store/token-dialog/token-dialog-actions';
import { openRepositoriesPanel } from "store/repositories/repositories-actions";
import {
    navigateToSiteManager,
    navigateToSshKeysUser,
    navigateToMyAccount,
    navigateToLinkAccount
} from 'store/navigation/navigation-action';
import { openUserVirtualMachines } from "store/virtual-machines/virtual-machines-actions";
import { pluginConfig } from 'plugins';
import { ElementListReducer } from 'common/plugintypes';

interface AccountMenuProps {
    user?: User;
    currentRoute: string;
    workbenchURL: string;
    apiToken?: string;
    localCluster: string;
}

const mapStateToProps = (state: RootState): AccountMenuProps => ({
    user: state.auth.user,
    currentRoute: state.router.location ? state.router.location.pathname : '',
    workbenchURL: state.auth.config.workbenchUrl,
    apiToken: state.auth.apiToken,
    localCluster: state.auth.localCluster
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

export const AccountMenuComponent =
    ({ user, dispatch, currentRoute, workbenchURL, apiToken, localCluster, classes }: AccountMenuProps & DispatchProp<any> & WithStyles<CssRules>) => {
        let accountMenuItems = <>
            <MenuItem onClick={() => dispatch(openUserVirtualMachines())}>Virtual Machines</MenuItem>
            <MenuItem onClick={() => dispatch(openRepositoriesPanel())}>Repositories</MenuItem>
            <MenuItem onClick={() => {
                dispatch<any>(getNewExtraToken(true));
                dispatch(openTokenDialog);
            }}>Get API token</MenuItem>
            <MenuItem onClick={() => dispatch(navigateToSshKeysUser)}>Ssh Keys</MenuItem>
            <MenuItem onClick={() => dispatch(navigateToSiteManager)}>Site Manager</MenuItem>
            <MenuItem onClick={() => dispatch(navigateToMyAccount)}>My account</MenuItem>
            <MenuItem onClick={() => dispatch(navigateToLinkAccount)}>Link account</MenuItem>
            <MenuItem>
                <a href={`${workbenchURL.replace(/\/$/, "")}/${wb1URL(currentRoute)}?api_token=${apiToken}`}
                    className={classes.link}>
                    Switch to Workbench v1</a></MenuItem>
        </>;

        const reduceItemsFn: (a: React.ReactElement[],
            b: ElementListReducer) => React.ReactElement[] = (a, b) => b(a);

        accountMenuItems = React.createElement(React.Fragment, null,
            pluginConfig.accountMenuList.reduce(reduceItemsFn, React.Children.toArray(accountMenuItems.props.children)));

        return user
            ? <DropdownMenu
                icon={<UserPanelIcon />}
                id="account-menu"
                title="Account Management"
                key={currentRoute}>
                <MenuItem disabled>
                    {getUserDisplayName(user)} {user.uuid.substr(0, 5) !== localCluster && `(${user.uuid.substr(0, 5)})`}
                </MenuItem>
                {user.isActive && accountMenuItems}
                <Divider />
                <MenuItem data-cy="logout-menuitem"
                    onClick={() => dispatch(authActions.LOGOUT({ deleteLinkData: true }))}>
                    Logout
		 </MenuItem>
            </DropdownMenu>
            : null;
    };

export const AccountMenu = withStyles(styles)(connect(mapStateToProps)(AccountMenuComponent));
