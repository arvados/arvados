// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Button, Typography } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { SshKeyResource } from '~/models/ssh-key';


type CssRules = 'root' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
       width: '100%'
    },
    link: {
        color: theme.palette.primary.main,
        textDecoration: 'none',
        margin: '0px 4px'
    }
});

export interface SshKeyPanelRootActionProps {
    onClick: () => void;
}

export interface SshKeyPanelRootDataProps {
    sshKeys?: SshKeyResource[];
}

type SshKeyPanelRootProps = SshKeyPanelRootDataProps & SshKeyPanelRootActionProps & WithStyles<CssRules>;

export const SshKeyPanelRoot = withStyles(styles)(
    ({ classes, sshKeys, onClick }: SshKeyPanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                <Typography variant='body1' paragraph={true}>
                    You have not yet set up an SSH public key for use with Arvados.
                    <a href='https://doc.arvados.org/user/getting_started/ssh-access-unix.html' target='blank' className={classes.link}>
                        Learn more.
                    </a>
                </Typography>
                <Typography variant='body1' paragraph={true}>
                    When you have an SSH key you would like to use, add it using button below.
                </Typography>
                <Button
                    onClick={onClick}
                    color="primary"
                    variant="contained">
                    Add New Ssh Key
                </Button>
            </CardContent>
        </Card>
    );