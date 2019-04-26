// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardContent,
    Button,
    Typography,
    Grid,
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { User } from "~/models/user";
import { LinkAccountType, AccountToLink } from "~/models/link-account";
import { formatDate }from "~/common/formatters";

type CssRules = 'root';// | 'gridItem' | 'label' | 'title' | 'actions';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    }
});

export interface LinkAccountPanelRootDataProps {
    user?: User;
    accountToLink?: AccountToLink;
}

export interface LinkAccountPanelRootActionProps {
    saveAccountLinkData: (type: LinkAccountType) => void;
    removeAccountLinkData: () => void;
}

type LinkAccountPanelRootProps = LinkAccountPanelRootDataProps & LinkAccountPanelRootActionProps & WithStyles<CssRules>;

export const LinkAccountPanelRoot = withStyles(styles) (
    ({classes, user, accountToLink, saveAccountLinkData, removeAccountLinkData}: LinkAccountPanelRootProps) => {
        return <Card className={classes.root}>
            <CardContent>
            { user && accountToLink===undefined && <Grid container spacing={24}>
                <Grid container item direction="column" spacing={24}>
                    <Grid item>
                        You are currently logged in as <b>{user.email}</b> ({user.username}, {user.uuid}) created on <b>{formatDate(user.createdAt)}</b>
                    </Grid>
                    <Grid item>
                        You can link Arvados accounts. After linking, either login will take you to the same account.
                    </Grid>
                </Grid>
                <Grid container item direction="row" spacing={24}>
                    <Grid item>
                        <Button color="primary" variant="contained" onClick={() => saveAccountLinkData(LinkAccountType.ADD_OTHER_LOGIN)}>
                            Add another login to this account
                        </Button>
                    </Grid>
                    <Grid item>
                        <Button color="primary" variant="contained" onClick={() => saveAccountLinkData(LinkAccountType.ACCESS_OTHER_ACCOUNT)}>
                            Use this login to access another account
                        </Button>
                    </Grid>
                </Grid>
            </Grid>}
            { accountToLink && <Grid container spacing={24}>
                <Grid container item direction="row" spacing={24}>
                    <Grid item>
                        <Button variant="contained" onClick={() => removeAccountLinkData()}>
                            Cancel
                        </Button>
                    </Grid>
                    <Grid item>
                        <Button color="primary" variant="contained" onClick={() => {}}>
                            Link accounts
                        </Button>
                    </Grid>
                </Grid>
            </Grid> }
            </CardContent>
        </Card>;
});