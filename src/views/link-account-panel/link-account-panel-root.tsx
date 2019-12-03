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
    Select,
    CircularProgress
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { UserResource } from "~/models/user";
import { LinkAccountType } from "~/models/link-account";
import { formatDate } from "~/common/formatters";
import { LinkAccountPanelStatus, LinkAccountPanelError } from "~/store/link-account-panel/link-account-panel-reducer";
import { Config } from '~/common/config';

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto',
        display: 'flex'
    }
});

export interface LinkAccountPanelRootDataProps {
    targetUser?: UserResource;
    userToLink?: UserResource;
    remoteHostsConfig: { [key: string]: Config };
    hasRemoteHosts: boolean;
    localCluster: string;
    loginCluster: string;
    status: LinkAccountPanelStatus;
    error: LinkAccountPanelError;
    selectedCluster?: string;
    isProcessing: boolean;
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
        const homeCluster = user.uuid.substr(0, 5);
        disp.push(<span> hosted on cluster <b>{homeCluster}</b> and </span>);
    }
    if (showCreatedAt) {
        disp.push(<span> created on <b>{formatDate(user.createdAt)}</b></span>);
    }
    return disp;
}

function isLocalUser(uuid: string, localCluster: string) {
    return uuid.substring(0, 5) === localCluster;
}

type LinkAccountPanelRootProps = LinkAccountPanelRootDataProps & LinkAccountPanelRootActionProps & WithStyles<CssRules>;

export const LinkAccountPanelRoot = withStyles(styles)(
    ({ classes, targetUser, userToLink, status, isProcessing, error, startLinking, cancelLinking, linkAccount,
        remoteHostsConfig, hasRemoteHosts, selectedCluster, setSelectedCluster, localCluster, loginCluster }: LinkAccountPanelRootProps) => {
        return <Card className={classes.root}>
            <CardContent>
                {isProcessing && <Grid container item direction="column" alignContent="center" spacing={24}>
                    <Grid item>
                        Loading user info. Please wait.
	                </Grid>
                    <Grid item style={{ alignSelf: 'center' }}>
                        <CircularProgress />
                    </Grid>
                </Grid>}
                {!isProcessing && status === LinkAccountPanelStatus.INITIAL && targetUser && <div>
                    {isLocalUser(targetUser.uuid, localCluster) ? <Grid container spacing={24}>
                        <Grid container item direction="column" spacing={24}>
                            <Grid item>
                                You are currently logged in as {displayUser(targetUser, true)}
                            </Grid>
                            <Grid item>
                                You can link Arvados accounts. After linking, either login will take you to the same account.
		                    </Grid >
                        </Grid>
                        <Grid container item direction="row" spacing={24}>
                            <Grid item>
                                <Button disabled={!targetUser.isActive} color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ADD_OTHER_LOGIN)}>
                                    Add another login to this account
			                    </Button>
                            </Grid>
                            <Grid item>
                                <Button color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ACCESS_OTHER_ACCOUNT)}>
                                    Use this login to access another account
			                    </Button>
                            </Grid>
                        </Grid>
                        {hasRemoteHosts && selectedCluster && <Grid container item direction="column" spacing={24}>
                            <Grid item>
                                You can also link {displayUser(targetUser, false)} with an account from a remote cluster.
		                    </Grid>
                            <Grid item>
                                Please select the cluster that hosts the account you want to link with:
                                <Select id="remoteHostsDropdown" native defaultValue={selectedCluster} style={{ marginLeft: "1em" }}
                                    onChange={(event) => setSelectedCluster(event.target.value)}>
                                    {Object.keys(remoteHostsConfig).map((k) => k !== localCluster ? <option key={k} value={k}>{k}</option> : null)}
                                </Select>
                            </Grid>
                            <Grid item>
                                <Button color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ACCESS_OTHER_REMOTE_ACCOUNT)}>
                                    Link with an account on&nbsp;{hasRemoteHosts ? <label>{selectedCluster} </label> : null}
                                </Button>
                            </Grid>
                        </Grid>}
                    </Grid> :
                        <Grid container spacing={24}>
                            <Grid container item direction="column" spacing={24}>
                                <Grid item>
                                    You are currently logged in as {displayUser(targetUser, true, true)}
                                </Grid>
                                {targetUser.isActive ?
                                    (loginCluster === "" ?
                                        <> <Grid item>
                                            This a remote account. You can link a local Arvados account to this one.
                                            After linking, you can access the local account's data by logging into the
					                        <b>{localCluster}</b> cluster as user <b>{targetUser.email}</b>
                                            from <b>{targetUser.uuid.substr(0, 5)}</b>.
					                    </Grid >
                                            <Grid item>
                                                <Button color="primary" variant="contained" onClick={() => startLinking(LinkAccountType.ADD_LOCAL_TO_REMOTE)}>
                                                    Link an account from {localCluster} to this account
					                            </Button>
                                            </Grid> </>
                                        : <Grid item>Please visit cluster
				                        <a href={remoteHostsConfig[loginCluster].workbench2Url + "/link_account"}>{loginCluster}</a>
                                            to perform account linking.</Grid>
                                    )
                                    : <Grid item>
                                        This an inactive remote account. An administrator must activate your
                                        account before you can proceed.  After your accounts is activated,
				                    you can link a local Arvados account hosted by the <b>{localCluster}</b>
                                        cluster to this one.
				                </Grid >}
                            </Grid>
                        </Grid>}
                </div>}
                {!isProcessing && (status === LinkAccountPanelStatus.LINKING || status === LinkAccountPanelStatus.ERROR) && userToLink && targetUser &&
                    <Grid container spacing={24}>
                        {status === LinkAccountPanelStatus.LINKING && <Grid container item direction="column" spacing={24}>
                            <Grid item>
                                Clicking 'Link accounts' will link {displayUser(userToLink, true, !isLocalUser(targetUser.uuid, localCluster))} to {displayUser(targetUser, true, !isLocalUser(targetUser.uuid, localCluster))}.
		                    </Grid>
                            {(isLocalUser(targetUser.uuid, localCluster)) && <Grid item>
                                After linking, logging in as {displayUser(userToLink)} will log you into the same account as {displayUser(targetUser)}.
		                    </Grid>}
                            <Grid item>
                                Any object owned by {displayUser(userToLink)} will be transfered to {displayUser(targetUser)}.
		                    </Grid>
                            {!isLocalUser(targetUser.uuid, localCluster) && <Grid item>
                                You can access <b>{userToLink.email}</b> data by logging into <b>{localCluster}</b> with the <b>{targetUser.email}</b> account.
		                    </Grid>}
                        </Grid>}
                        {error === LinkAccountPanelError.NON_ADMIN && <Grid item>
                            Cannot link admin account {displayUser(userToLink)} to non-admin account {displayUser(targetUser)}.
		                </Grid>}
                        {error === LinkAccountPanelError.SAME_USER && <Grid item>
                            Cannot link {displayUser(targetUser)} to the same account.
		                </Grid>}
                        {error === LinkAccountPanelError.INACTIVE && <Grid item>
                            Cannot link account {displayUser(userToLink)} to inactive account {displayUser(targetUser)}.
		                </Grid>}
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
                    </Grid>}
            </CardContent>
        </Card>;
    });
