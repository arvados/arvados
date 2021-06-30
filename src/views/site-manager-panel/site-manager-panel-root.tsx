// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Card,
    CardContent,
    CircularProgress,
    Grid,
    IconButton,
    StyleRulesCallback,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    Typography,
    WithStyles,
    withStyles
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { Session, SessionStatus } from "models/session";
import Button from "@material-ui/core/Button";
import { compose, Dispatch } from "redux";
import { Field, FormErrors, InjectedFormProps, reduxForm, reset, stopSubmit } from "redux-form";
import { TextField } from "components/text-field/text-field";
import { addSession } from "store/auth/auth-action-session";
import { SITE_MANAGER_REMOTE_HOST_VALIDATION } from "validators/validators";
import { Config } from 'common/config';
import { ResourceCluster } from 'views-components/data-explorer/renderers';
import { TrashIcon } from "components/icon/icon";

type CssRules = 'root' | 'link' | 'buttonContainer' | 'table' | 'tableRow' |
    'remoteSiteInfo' | 'buttonAdd' | 'buttonLoggedIn' | 'buttonLoggedOut' |
    'statusCell';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    link: {
        color: theme.palette.primary.main,
        textDecoration: 'none',
        margin: '0px 4px'
    },
    buttonContainer: {
        textAlign: 'right'
    },
    table: {
        marginTop: theme.spacing.unit
    },
    tableRow: {
        '& td, th': {
            whiteSpace: 'nowrap'
        }
    },
    statusCell: {
        minWidth: 160
    },
    remoteSiteInfo: {
        marginTop: 20
    },
    buttonAdd: {
        marginLeft: 10,
        marginTop: theme.spacing.unit * 3
    },
    buttonLoggedIn: {
        minHeight: theme.spacing.unit,
        padding: 5,
        color: '#fff',
        backgroundColor: '#009966',
        '&:hover': {
            backgroundColor: '#008450',
        }
    },
    buttonLoggedOut: {
        minHeight: theme.spacing.unit,
        padding: 5,
        color: '#000',
        backgroundColor: '#FFC414',
        '&:hover': {
            backgroundColor: '#eaaf14',
        }
    }
});

export interface SiteManagerPanelRootActionProps {
    toggleSession: (session: Session) => void;
    removeSession: (session: Session) => void;
}

export interface SiteManagerPanelRootDataProps {
    sessions: Session[];
    remoteHostsConfig: { [key: string]: Config };
    localClusterConfig: Config;
}

type SiteManagerPanelRootProps = SiteManagerPanelRootDataProps & SiteManagerPanelRootActionProps & WithStyles<CssRules> & InjectedFormProps;
const SITE_MANAGER_FORM_NAME = 'siteManagerForm';

const submitSession = (remoteHost: string) =>
    (dispatch: Dispatch) => {
        dispatch<any>(addSession(remoteHost, undefined, true)).then(() => {
            dispatch(reset(SITE_MANAGER_FORM_NAME));
        }).catch((e: any) => {
            const errors = {
                remoteHost: e
            } as FormErrors;
            dispatch(stopSubmit(SITE_MANAGER_FORM_NAME, errors));
        });
    };

export const SiteManagerPanelRoot = compose(
    reduxForm<{ remoteHost: string }>({
        form: SITE_MANAGER_FORM_NAME,
        touchOnBlur: false,
        onSubmit: (data, dispatch) => {
            dispatch(submitSession(data.remoteHost));
        }
    }),
    withStyles(styles))
    (({ classes, sessions, handleSubmit, toggleSession, removeSession, localClusterConfig, remoteHostsConfig }: SiteManagerPanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                <Grid container direction="row">
                    <Grid item xs={12}>
                        <Typography paragraph={true} >
                            You can log in to multiple Arvados sites here, then use the multi-site search page to search collections and projects on all sites at once.
		    </Typography>
                    </Grid>
                </Grid>
                <Grid item xs={12}>
                    {sessions.length > 0 && <Table className={classes.table}>
                        <TableHead>
                            <TableRow className={classes.tableRow}>
                                <TableCell>Cluster ID</TableCell>
                                <TableCell>Host</TableCell>
                                <TableCell>Email</TableCell>
                                <TableCell>UUID</TableCell>
                                <TableCell>Status</TableCell>
                                <TableCell>Actions</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {sessions.map((session, index) => {
                                const validating = session.status === SessionStatus.BEING_VALIDATED;
                                return <TableRow key={index} className={classes.tableRow}>
                                    <TableCell>{remoteHostsConfig[session.clusterId] ?
                                        <a href={remoteHostsConfig[session.clusterId].workbench2Url} style={{ textDecoration: 'none' }}> <ResourceCluster uuid={session.clusterId} /></a>
                                        : session.clusterId}</TableCell>
                                    <TableCell>{session.remoteHost}</TableCell>
                                    <TableCell>{validating ? <CircularProgress size={20} /> : session.email}</TableCell>
                                    <TableCell>{validating ? <CircularProgress size={20} /> : session.uuid}</TableCell>
                                    <TableCell className={classes.statusCell}>
                                        <Button fullWidth
                                            disabled={validating || session.status === SessionStatus.INVALIDATED || session.active}
                                            className={session.loggedIn ? classes.buttonLoggedIn : classes.buttonLoggedOut}
                                            onClick={() => toggleSession(session)}>
                                            {validating ? "Validating"
                                                : (session.loggedIn ?
                                                    (session.userIsActive ? "Logged in" : "Inactive")
                                                    : "Logged out")}
                                        </Button>
                                    </TableCell>
                                    <TableCell>
                                        {session.clusterId !== localClusterConfig.uuidPrefix &&
                                            !localClusterConfig.clusterConfig.RemoteClusters[session.clusterId] &&
                                            <IconButton onClick={() => removeSession(session)}>
                                                <TrashIcon />
                                            </IconButton>}
                                    </TableCell>
                                </TableRow>;
                            })}
                        </TableBody>
                    </Table>}
                </Grid>
                <form onSubmit={handleSubmit}>
                    <Grid container direction="row">
                        <Grid item xs={12}>
                            <Typography paragraph={true} className={classes.remoteSiteInfo}>
                                To add a remote Arvados site, paste the remote site's host here (see "ARVADOS_API_HOST" on the "current token" page).
                        </Typography>
                        </Grid>
                        <Grid item xs={8}>
                            <Field
                                name='remoteHost'
                                validate={SITE_MANAGER_REMOTE_HOST_VALIDATION}
                                component={TextField}
                                placeholder="zzzz.arvadosapi.com"
                                margin="normal"
                                label="New cluster"
                                autoFocus />
                        </Grid>
                        <Grid item xs={3}>
                            <Button type="submit" variant="contained" color="primary"
                                className={classes.buttonAdd}>
                                {"ADD"}</Button>
                        </Grid>
                    </Grid>
                </form>
            </CardContent>
        </Card>
    );
