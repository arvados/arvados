// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { MenuItem } from "@mui/material";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { User, getUserDisplayName } from "models/user";
import { DropdownMenu } from "components/dropdown-menu/dropdown-menu";
import { UserPanelIcon } from "components/icon/icon";
import { connect } from 'react-redux';
import { authActions, getNewExtraToken } from 'store/auth/auth-action';
import { RootState } from "store/store";
import { openTokenDialog } from 'store/token-dialog/token-dialog-actions';
import {
    navigateToSiteManager,
    navigateToSshKeysUser,
    navigateToMyAccount,
    navigateToLinkAccount,
} from 'store/navigation/navigation-action';
import { pluginConfig } from 'plugins';
import { ElementListReducer } from 'common/plugintypes';
import { Dispatch } from 'redux';

interface AccountMenuProps {
    user?: User;
    currentRoute: string;
    workbenchURL: string;
    apiToken?: string;
    localCluster: string;
}

interface AccountMenuActionProps {
    onLogout: () => void;
    getNewExtraToken: (reuseExtra?: boolean) => void;
    openTokenDialog: () => void;
    navigateToSshKeysUser: () => void;
    navigateToSiteManager: () => void;
    navigateToMyAccount: () => void;
    navigateToLinkAccount: () => void;
}

const mapStateToProps = (state: RootState): AccountMenuProps => ({
    user: state.auth.user,
    currentRoute: state.router.location ? state.router.location.pathname : '',
    workbenchURL: state.auth.config.workbenchUrl,
    apiToken: state.auth.apiToken,
    localCluster: state.auth.localCluster
});

const mapDispatchToProps = (dispatch: Dispatch): AccountMenuActionProps => ({
    onLogout: () => {
        dispatch<any>(dispatch(authActions.LOGOUT({ deleteLinkData: true, preservePath: false })));
    },
    getNewExtraToken: (reuseExtra: boolean) => dispatch<any>(getNewExtraToken(reuseExtra)),
    openTokenDialog: () => dispatch<any>(openTokenDialog),
    navigateToSshKeysUser: () => dispatch<any>(navigateToSshKeysUser),
    navigateToSiteManager: () => dispatch<any>(navigateToSiteManager),
    navigateToMyAccount: () => dispatch<any>(navigateToMyAccount),
    navigateToLinkAccount: () => dispatch<any>(navigateToLinkAccount),
});

type CssRules = 'link';

const styles: CustomStyleRulesCallback<CssRules> = () => ({
    link: {
        textDecoration: 'none',
        color: 'inherit'
    }
});

export const AccountMenuComponent =
    ({ user, currentRoute, localCluster, onLogout, getNewExtraToken, openTokenDialog, navigateToSshKeysUser, navigateToSiteManager, navigateToMyAccount, navigateToLinkAccount }: AccountMenuProps & AccountMenuActionProps & WithStyles<CssRules>) => {
        
        console.log(user, currentRoute, localCluster, onLogout, getNewExtraToken, openTokenDialog, navigateToSshKeysUser, navigateToSiteManager, navigateToMyAccount, navigateToLinkAccount);
        
        let accountMenuItems = <>
            <MenuItem onClick={() => {
                getNewExtraToken(true);
                openTokenDialog();
            }}>Get API token</MenuItem>
            <MenuItem onClick={navigateToSshKeysUser}>SSH Keys</MenuItem>
            <MenuItem onClick={navigateToSiteManager}>Site Manager</MenuItem>
            <MenuItem onClick={navigateToMyAccount}>My account</MenuItem>
            <MenuItem onClick={navigateToLinkAccount}>Link account</MenuItem>
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
                    {getUserDisplayName(user)} {user.uuid.substring(0, 5) !== localCluster && `(${user.uuid.substring(0, 5)})`}
                </MenuItem>
                {user.isActive && accountMenuItems}
                <MenuItem data-cy="logout-menuitem"
                    onClick={onLogout}
                    >
                    Logout
                </MenuItem>
            </DropdownMenu>
            : null;
    };

export const AccountMenu = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(AccountMenuComponent));
