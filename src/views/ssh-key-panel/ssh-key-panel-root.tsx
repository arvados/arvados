// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Button, Typography, Grid, Table, TableHead, TableRow, TableCell, TableBody, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { SshKeyResource } from '~/models/ssh-key';
import { AddIcon, MoreOptionsIcon, KeyIcon } from '~/components/icon/icon';

type CssRules = 'root' | 'link' | 'buttonContainer' | 'table' | 'tableRow' | 'keyIcon';

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
    keyIcon: {
        color: theme.palette.primary.main
    }
});

export interface SshKeyPanelRootActionProps {
    openSshKeyCreateDialog: () => void;
    openRowOptions: (event: React.MouseEvent<HTMLElement>, sshKey: SshKeyResource) => void;
    openPublicKeyDialog: (name: string, publicKey: string) => void;
}

export interface SshKeyPanelRootDataProps {
    sshKeys: SshKeyResource[];
    hasKeys: boolean;
}

type SshKeyPanelRootProps = SshKeyPanelRootDataProps & SshKeyPanelRootActionProps & WithStyles<CssRules>;

export const SshKeyPanelRoot = withStyles(styles)(
    ({ classes, sshKeys, openSshKeyCreateDialog, openPublicKeyDialog, hasKeys, openRowOptions }: SshKeyPanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                <Grid container direction="row">
                    <Grid item xs={8}>
                        { !hasKeys && <Typography variant='body1' paragraph={true} >
                            You have not yet set up an SSH public key for use with Arvados.
                            <a href='https://doc.arvados.org/user/getting_started/ssh-access-unix.html'
                                target='blank' className={classes.link}>
                                Learn more.
                            </a>
                        </Typography>}
                        { !hasKeys && <Typography variant='body1' paragraph={true}>
                            When you have an SSH key you would like to use, add it using button below.
                        </Typography> }
                    </Grid>
                    <Grid item xs={4} className={classes.buttonContainer}>
                        <Button onClick={openSshKeyCreateDialog} color="primary" variant="contained">
                            <AddIcon /> Add New Ssh Key
                        </Button>
                    </Grid>
                </Grid>
                <Grid item xs={12}>
                    {hasKeys && <Table className={classes.table}>
                        <TableHead>
                            <TableRow className={classes.tableRow}>
                                <TableCell>Name</TableCell>
                                <TableCell>UUID</TableCell>
                                <TableCell>Authorized user</TableCell>
                                <TableCell>Expires at</TableCell>
                                <TableCell>Key type</TableCell>
                                <TableCell>Public Key</TableCell>
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {sshKeys.map((sshKey, index) =>
                                <TableRow key={index} className={classes.tableRow}>
                                    <TableCell>{sshKey.name}</TableCell>
                                    <TableCell>{sshKey.uuid}</TableCell>
                                    <TableCell>{sshKey.authorizedUserUuid}</TableCell>
                                    <TableCell>{sshKey.expiresAt || '(none)'}</TableCell>
                                    <TableCell>{sshKey.keyType}</TableCell>
                                    <TableCell>
                                        <Tooltip title="Public Key" disableFocusListener>
                                            <IconButton onClick={() => openPublicKeyDialog(sshKey.name, sshKey.publicKey)}>
                                                <KeyIcon className={classes.keyIcon} />
                                            </IconButton>
                                        </Tooltip>
                                    </TableCell>
                                    <TableCell>
                                        <Tooltip title="More options" disableFocusListener>
                                            <IconButton onClick={event => openRowOptions(event, sshKey)}>
                                                <MoreOptionsIcon />
                                            </IconButton>
                                        </Tooltip>
                                    </TableCell>
                                </TableRow>)}
                        </TableBody>
                    </Table>}
                </Grid>
            </CardContent>
        </Card>
    );