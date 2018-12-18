// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    Card,
    CardContent, CircularProgress,
    Grid,
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
import { ArvadosTheme } from '~/common/custom-theme';
import { Session, SessionStatus } from "~/models/session";
import Button from "@material-ui/core/Button";
import { User } from "~/models/user";
import { compose } from "redux";
import { Field, FormErrors, InjectedFormProps, reduxForm, reset, stopSubmit } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { addSession } from "~/store/auth/auth-action-session";
import { SITE_MANAGER_REMOTE_HOST_VALIDATION } from "~/validators/validators";
import {
    RENAME_FILE_DIALOG,
    RenameFileDialogData
} from "~/store/collection-panel/collection-panel-files/collection-panel-files-actions";

type CssRules = 'root' | 'link' | 'buttonContainer' | 'table' | 'tableRow' | 'status' | 'remoteSiteInfo' | 'buttonAdd';

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
    status: {
        width: 100,
        padding: 5,
        fontWeight: 'bold',
        textAlign: 'center',
        borderRadius: 4
    },
    remoteSiteInfo: {
        marginTop: 20
    },
    buttonAdd: {
        marginLeft: 10,
        marginTop: theme.spacing.unit * 3
    }
});

export interface SiteManagerPanelRootActionProps {
}

export interface SiteManagerPanelRootDataProps {
    sessions: Session[];
    user: User;
}

type SiteManagerPanelRootProps = SiteManagerPanelRootDataProps & SiteManagerPanelRootActionProps & WithStyles<CssRules> & InjectedFormProps;
const SITE_MANAGER_FORM_NAME = 'siteManagerForm';

export const SiteManagerPanelRoot = compose(
    reduxForm<{remoteHost: string}>({
        form: SITE_MANAGER_FORM_NAME,
        onSubmit: async (data, dispatch) => {
            try {
                await dispatch(addSession(data.remoteHost));
                dispatch(reset(SITE_MANAGER_FORM_NAME));
            } catch (e) {
                const errors = {
                    remoteHost: e
                } as FormErrors;
                dispatch(stopSubmit(SITE_MANAGER_FORM_NAME, errors));
            }

        }
    }),
    withStyles(styles))
    (({ classes, sessions, handleSubmit }: SiteManagerPanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                <Grid container direction="row">
                    <Grid item xs={12}>
                        <Typography variant='body1' paragraph={true} >
                            You can log in to multiple Arvados sites here, then use the multi-site search page to search collections and projects on all sites at once.
                        </Typography>
                    </Grid>
                </Grid>
                <Grid item xs={12}>
                    {sessions.length > 0 && <Table className={classes.table}>
                        <TableHead>
                            <TableRow className={classes.tableRow}>
                                <TableCell>Cluster ID</TableCell>
                                <TableCell>Username</TableCell>
                                <TableCell>Email</TableCell>
                                <TableCell>Status</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {sessions.map((session, index) => {
                                const validating = session.status === SessionStatus.BEING_VALIDATED;
                                return <TableRow key={index} className={classes.tableRow}>
                                    <TableCell>{session.clusterId}</TableCell>
                                    <TableCell>{validating ? <CircularProgress size={20}/> : session.username}</TableCell>
                                    <TableCell>{validating ? <CircularProgress size={20}/> : session.email}</TableCell>
                                    <TableCell>
                                        <div className={classes.status} style={{
                                            color: session.loggedIn ? '#fff' : '#000',
                                            backgroundColor: session.loggedIn ? '#009966' : '#FFC414'
                                        }}>
                                            {session.loggedIn ? "Logged in" : "Logged out"}
                                        </div>
                                    </TableCell>
                                </TableRow>;
                            })}
                        </TableBody>
                    </Table>}
                </Grid>
                <form onSubmit={handleSubmit}>
                    <Grid container direction="row">
                        <Grid item xs={12}>
                            <Typography variant='body1' paragraph={true} className={classes.remoteSiteInfo}>
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
                                autoFocus/>
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
