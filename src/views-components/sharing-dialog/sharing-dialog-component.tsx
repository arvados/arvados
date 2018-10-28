// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dialog, DialogTitle, Button, Grid, DialogContent } from '@material-ui/core';
import { DialogActions } from '~/components/dialog-actions/dialog-actions';


export interface SharingDialogDataProps {
    open: boolean;
    saveEnabled: boolean;
    advancedEnabled: boolean;
    children: React.ReactNode;
}
export interface SharingDialogActionProps {
    onClose: () => void;
    onExited: () => void;
    onSave: () => void;
    onAdvanced: () => void;
}
export default (props: SharingDialogDataProps & SharingDialogActionProps) => {
    const { children, open, advancedEnabled, saveEnabled, onAdvanced, onClose, onExited, onSave } = props;
    return <Dialog
        {...{ open, onClose, onExited }}
        fullWidth
        maxWidth='sm'>
        <DialogTitle>
            Sharing settings
            </DialogTitle>
        <DialogContent>
            {children}
        </DialogContent>
        <DialogActions>
            <Grid container spacing={8}>
                {advancedEnabled &&
                    <Grid item>
                        <Button
                            color='primary'
                            onClick={onAdvanced}>
                            Advanced
                    </Button>
                    </Grid>
                }
                <Grid item xs />
                <Grid item>
                    <Button onClick={onClose}>
                        Close
                    </Button>
                </Grid>
                <Grid item>
                    <Button
                        variant='contained'
                        color='primary'
                        onClick={onSave}
                        disabled={!saveEnabled}>
                        Save
                    </Button>
                </Grid>
            </Grid>
        </DialogActions>
    </Dialog>;
};
