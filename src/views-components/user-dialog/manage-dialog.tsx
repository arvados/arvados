// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from "redux";
import { USER_MANAGE_DIALOG } from "~/store/users/users-actions";
import { UserResource } from "~/models/user";

type CssRules = 'spacing';

const styles = withStyles<CssRules>((theme: ArvadosTheme) => ({
    spacing: {
        paddingBottom: theme.spacing.unit * 2,
        paddingTop: theme.spacing.unit * 2,
    }
}));

interface UserManageDataProps {
    data: UserResource;
}

type UserManageProps = UserManageDataProps & WithStyles<CssRules>;

export const UserManageDialog = compose(
    withDialog(USER_MANAGE_DIALOG),
    styles)(
        (props: WithDialogProps<UserManageProps> & UserManageProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth="md">
                <DialogTitle>{`Manage - ${props.data.firstName} ${props.data.lastName}`}</DialogTitle>
                <DialogContent>
                    <Typography variant="body2" className={props.classes.spacing}>
                        As an admin, you can log in as this user. When you’ve finished, you will need to log out and log in again with your own account.
                    </Typography>
                    <Button variant="contained" color="primary">
                        {`LOG IN AS ${props.data.firstName} ${props.data.lastName}`}
                    </Button>
                    <Typography variant="body2" className={props.classes.spacing}>
                        As an admin, you can setup a shell account for this user. The login name is automatically generated from the user's e-mail address.
                    </Typography>
                    <Button variant="contained" color="primary">
                        {`SETUP SHELL ACCOUNT FOR ${props.data.firstName} ${props.data.lastName}`}
                    </Button>
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='flat'
                        color='primary'
                        onClick={props.closeDialog}>
                        Close
                </Button>
                </DialogActions>
            </Dialog>
    );
