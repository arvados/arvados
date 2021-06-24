// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, DialogContentText } from "@material-ui/core";
import { WithDialogProps } from "store/dialog/with-dialog";
import { WarningIcon } from 'components/icon/icon';

export interface ConfirmationDialogDataProps {
    title: string;
    text: string;
    info?: string;
    cancelButtonLabel?: string;
    confirmButtonLabel?: string;
}

export interface ConfirmationDialogProps {
    onConfirm: () => void;
}

export const ConfirmationDialog = (props: ConfirmationDialogProps & WithDialogProps<ConfirmationDialogDataProps>) =>
    <Dialog open={props.open}>
        <div data-cy='confirmation-dialog'>
            <DialogTitle>{props.data.title}</DialogTitle>
            <DialogContent style={{ display: 'flex', alignItems: 'center' }}>
                <WarningIcon />
                <DialogContentText style={{ paddingLeft: '8px' }}>
                    <div>{props.data.text}</div>
                    <div>{props.data.info}</div>
                </DialogContentText>
            </DialogContent>
            <DialogActions style={{ margin: '0px 24px 24px' }}>
                <Button
                    data-cy='confirmation-dialog-cancel-btn'
                    variant='text'
                    color='primary'
                    onClick={props.closeDialog}>
                    {props.data.cancelButtonLabel || 'Cancel'}
                </Button>
                <Button
                    data-cy='confirmation-dialog-ok-btn'
                    variant='contained'
                    color='primary'
                    type='submit'
                    onClick={props.onConfirm}>
                    {props.data.confirmButtonLabel || 'Ok'}
                </Button>
            </DialogActions>
        </div>
    </Dialog>;
