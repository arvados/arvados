// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, DialogContentText } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { WarningIcon } from '~/components/icon/icon';

export interface ConfirmationDialogDataProps {
    title: string;
    text: string;
    cancelButtonLabel?: string;
    confirmButtonLabel?: string;
}

export interface ConfirmationDialogProps {
    onConfirm: () => void;
}

export const ConfirmationDialog = (props: ConfirmationDialogProps & WithDialogProps<ConfirmationDialogDataProps>) =>
    <Dialog open={props.open}>
        <DialogTitle>{props.data.title}</DialogTitle>
        <DialogContent style={{ display: 'flex', alignItems: 'center' }}>
            <WarningIcon />
            <DialogContentText style={{ paddingLeft: '8px' }}>
                {props.data.text}
                <br />
                {props.data.title === 'Removing file' ? 'Removing a file will change content adress.' : 'Removing files will change content adress.'}
            </DialogContentText>
        </DialogContent>
        <DialogActions>
            <Button
                variant='flat'
                color='primary'
                onClick={props.closeDialog}>
                {props.data.cancelButtonLabel || 'Cancel'}
            </Button>
            <Button
                variant='contained'
                color='primary'
                type='submit'
                onClick={props.onConfirm}>
                {props.data.confirmButtonLabel || 'Ok'}
            </Button>
        </DialogActions>
    </Dialog>;
