// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
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
                    <span style={{display: 'block'}}>{props.data.text}</span>
                    <span style={{display: 'block'}}>{props.data.info}</span>
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
