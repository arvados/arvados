// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose, Dispatch } from "redux";
import { USER_MANAGEMENT_DIALOG, openSetupShellAccount, loginAs } from "~/store/users/users-actions";
import { connect } from "react-redux";

type CssRules = 'spacing';

const styles = withStyles<CssRules>((theme: ArvadosTheme) => ({
    spacing: {
        paddingBottom: theme.spacing.unit * 2,
        paddingTop: theme.spacing.unit * 2,
    }
}));

interface UserManageDataProps {
    data: any;
}

interface UserManageActionProps {
    openSetupShellAccount: (uuid: string) => void;
    loginAs: (uuid: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openSetupShellAccount: (uuid: string) => dispatch<any>(openSetupShellAccount(uuid)),
    loginAs: (uuid: string) => dispatch<any>(loginAs(uuid))
});

type UserManageProps = UserManageDataProps & UserManageActionProps & WithStyles<CssRules>;

export const UserManageDialog = compose(
    connect(null, mapDispatchToProps),
    withDialog(USER_MANAGEMENT_DIALOG),
    styles)(
        (props: WithDialogProps<UserManageProps> & UserManageProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth="md">
                {props.data &&
                    <span>
                        <DialogTitle>{`Manage - ${props.data.firstName} ${props.data.lastName}`}</DialogTitle>
                        <DialogContent>
                            <Typography variant='body1' className={props.classes.spacing}>
                                As an admin, you can log in as this user. When you’ve finished, you will need to log out and log in again with your own account.
                    </Typography>
                            <Button variant="contained" color="primary" onClick={() => props.loginAs(props.data.uuid)}>
                                {`LOG IN AS ${props.data.firstName} ${props.data.lastName}`}
                            </Button>
                            <Typography variant='body1' className={props.classes.spacing}>
                                As an admin, you can setup a shell account for this user. The login name is automatically generated from the user's e-mail address.
                    </Typography>
                            <Button variant="contained" color="primary" onClick={() => props.openSetupShellAccount(props.data.uuid)}>
                                {`SETUP SHELL ACCOUNT FOR ${props.data.firstName} ${props.data.lastName}`}
                            </Button>
                        </DialogContent></span>}

                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={props.closeDialog}>
                        Close
                </Button>
                </DialogActions>
            </Dialog>
    );
