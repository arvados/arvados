// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogActions, Button, CardHeader, DialogContent } from '@material-ui/core';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { withDialog } from "~/store/dialog/with-dialog";
import { PROCESS_INPUT_DIALOG_NAME, ProcessInputDialogData } from '~/store/processes/process-input-actions';

export const ProcessInputDialog = withDialog(PROCESS_INPUT_DIALOG_NAME)(
    (props: WithDialogProps<ProcessInputDialogData>) =>
        <Dialog
            open={props.open}
            maxWidth={false}
            onClose={props.closeDialog}>
            <CardHeader
                title="Inputs - Pipeline template that generates a config file from a template" />
            <DialogContent>
                cos
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