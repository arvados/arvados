// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button } from "@material-ui/core";
import { withDialog } from "store/dialog/with-dialog";
import { dialogActions } from "store/dialog/dialog-actions";

export const REMOVE_DIALOG = 'removeCollectionFilesDialog';

export const RemoveDialog = withDialog(REMOVE_DIALOG)(
    (props) =>
        <Dialog open={props.open}>
            <DialogTitle>{`Removing ${props.data}`}</DialogTitle>
            <DialogContent>
                {`Are you sure you want to remove ${props.data}?`}
            </DialogContent>
            <DialogActions>
                <Button
                    variant='text'
                    color='primary'
                    onClick={props.closeDialog}>
                    Cancel
                </Button>
                <Button variant='contained' color='primary'>
                    Remove
                </Button>
            </DialogActions>
        </Dialog>
);

export const openRemoveDialog = (removedDataName: string) =>
    dialogActions.OPEN_DIALOG({ id: REMOVE_DIALOG, data: removedDataName });
