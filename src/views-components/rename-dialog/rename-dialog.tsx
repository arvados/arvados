// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, Typography } from "@material-ui/core";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";

export const RENAME_DIALOG = 'nameDialog';

export const RenameDialog = withDialog(RENAME_DIALOG)(
    (props) =>
        <Dialog open={props.open}>
            <DialogTitle>{`Rename`}</DialogTitle>
            <DialogContent>
                <Typography variant='body1' gutterBottom>
                    {`Please, enter a new name for ${props.data}`}
                </Typography>
                <TextField fullWidth={true} placeholder='New name' />
            </DialogContent>
            <DialogActions>
                <Button
                    variant='flat'
                    color='primary'
                    onClick={props.closeDialog}>
                    Cancel
                </Button>
                <Button variant='raised' color='primary'>
                    Ok
                </Button>
            </DialogActions>
        </Dialog>
);

export const openRenameDialog = (originalName: string, ) =>
    dialogActions.OPEN_DIALOG({ id: RENAME_DIALOG, data: originalName });
