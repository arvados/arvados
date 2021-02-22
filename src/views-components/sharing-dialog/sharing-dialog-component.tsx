// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dialog, DialogTitle, Button, Grid, DialogContent, CircularProgress, Paper } from '@material-ui/core';
import { DialogActions } from '~/components/dialog-actions/dialog-actions';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';


export interface SharingDialogDataProps {
    open: boolean;
    loading: boolean;
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
    const { children, open, loading, advancedEnabled, saveEnabled, onAdvanced, onClose, onExited, onSave } = props;
    return <Dialog
        {...{ open, onClose, onExited }}
        className="sharing-dialog"
        fullWidth
        maxWidth='sm'
        disableBackdropClick
        disableEscapeKeyDown>
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
        {
            loading && <LoadingIndicator />
        }
    </Dialog>;
};

const loadingIndicatorStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        left: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'rgba(255, 255, 255, 0.8)',
    },
});

const LoadingIndicator = withStyles(loadingIndicatorStyles)(
    (props: WithStyles<'root'>) =>
        <Paper classes={props.classes}>
            <CircularProgress />
        </Paper>
);
