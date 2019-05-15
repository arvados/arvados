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
    Grid,
    Select
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { UserResource } from "~/models/user";
import { LinkAccountType } from "~/models/link-account";
import { formatDate } from "~/common/formatters";
import { LinkAccountPanelStatus, LinkAccountPanelError } from "~/store/link-account-panel/link-account-panel-reducer";

type CssRules = 'root';// | 'gridItem' | 'label' | 'title' | 'actions';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    }
});

export interface LinkAccountPanelRootDataProps {
    targetUser?: UserResource;
    userToLink?: UserResource;
    remoteHosts:  { [key: string]: string };
    hasRemoteHosts: boolean;
    status : LinkAccountPanelStatus;
    error: LinkAccountPanelError;
    selectedCluster?: string;
}

export interface LinkAccountPanelRootActionProps {
    startLinking: (type: LinkAccountType) => void;
    cancelLinking: () => void;
    linkAccount: () => void;
    setSelectedCluster: (cluster: string) => void;
}

function displayUser(user: UserResource, showCreatedAt: boolean = false, showCluster: boolean = false) {
    const disp = [];
    disp.push(<span><b>{user.email}</b> ({user.username}, {user.uuid})</span>);
    if (showCluster) {
        const homeCluster = user.uuid.substr(0,5);
        disp.push(<span> hosted on cluster <b>{homeCluster}</b> and </span>);
    }
    if (showCreatedAt) {
        disp.push(<span> created on <b>{formatDate(user.createdAt, true)}</b></span>);
    }
    return disp;
}

type LinkAccountPanelRootProps = LinkAccountPanelRootDataProps & LinkAccountPanelRootActionProps & WithStyles<CssRules>;

export const LinkAccountPanelRoot = withStyles(styles) (
    ({classes, targetUser, userToLink, status, error, startLinking, cancelLinking, linkAccount,
      remoteHosts, hasRemoteHosts, selectedCluster, setSelectedCluster}: LinkAccountPanelRootProps) => {
        return <Card className={classes.root}>
            <CardContent>
            { status === LinkAccountPanelStatus.INITIAL && targetUser &&
            <Grid container spacing={24}>
                <Grid container item direction="column" spacing={24}>
                    <Grid item>
                        You are currently logged in as {displayUser(targetUser, true, hasRemoteHosts)}
                    </Grid>
                    <Grid item>
                        You can link Arvados accounts. After linking, either login will take you to the same account.
                    </Grid >
                    {hasRemoteHosts && selectedCluster && <Grid item>
                        Please select the cluster that hosts the account you want to link with:
                            <Select id="remoteHostsDropdown" native defaultValue={selectedCluster} style={{ marginLeft: "1em" }}
                                onChange={(event) => setSelectedCluster(event.target.value)}>
                                {Object.keys(remoteHosts).map((k) => <option key={k} value={k}>{k}</option>)}
                            </Select>
                    </Grid> }
                </Grid>
                <Grid container item direction="row" spacing={24}>
                    <Grid item>
                        <Button disabled={!targetUser.isActive} color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ADD_OTHER_LOGIN)}>
                            Add another login&nbsp;{hasRemoteHosts ? <label> from {selectedCluster} </label> : null}&nbsp;to this account
                        </Button>
                    </Grid>
                    <Grid item>
                        <Button color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ACCESS_OTHER_ACCOUNT)}>
                            Use this login to access another account&nbsp;{hasRemoteHosts ? <label> on {selectedCluster} </label> : null}
                        </Button>
                    </Grid>
                </Grid>
            </Grid> }
            { (status === LinkAccountPanelStatus.LINKING || status === LinkAccountPanelStatus.ERROR) && userToLink && targetUser &&
            <Grid container spacing={24}>
                { status === LinkAccountPanelStatus.LINKING && <Grid container item direction="column" spacing={24}>
                    <Grid item>
                        Clicking 'Link accounts' will link {displayUser(userToLink, true, hasRemoteHosts)} to {displayUser(targetUser, true, hasRemoteHosts)}.
                    </Grid>
                    <Grid item>
                        After linking, logging in as {displayUser(userToLink)} will log you into the same account as {displayUser(targetUser)}.
                    </Grid>
                    <Grid item>
                       Any object owned by {displayUser(userToLink)} will be transfered to {displayUser(targetUser)}.
                    </Grid>
                </Grid> }
                { error === LinkAccountPanelError.NON_ADMIN && <Grid item>
                    Cannot link admin account {displayUser(userToLink)} to non-admin account {displayUser(targetUser)}.
                </Grid> }
                { error === LinkAccountPanelError.SAME_USER && <Grid item>
                    Cannot link {displayUser(targetUser)} to the same account.
                </Grid> }
                { error === LinkAccountPanelError.INACTIVE && <Grid item>
                    Cannot link account {displayUser(userToLink)} to inactive account {displayUser(targetUser)}.
                </Grid> }
                <Grid container item direction="row" spacing={24}>
                    <Grid item>
                        <Button variant="contained" onClick={() => cancelLinking()}>
                            Cancel
                        </Button>
                    </Grid>
                    <Grid item>
                        <Button disabled={status === LinkAccountPanelStatus.ERROR} color="primary" variant="contained" onClick={() => linkAccount()}>
                            Link accounts
                        </Button>
                    </Grid>
                </Grid>
            </Grid> }
            </CardContent>
        </Card> ;
});