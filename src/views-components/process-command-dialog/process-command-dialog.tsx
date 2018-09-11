// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogActions, Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { withDialog } from "~/store/dialog/with-dialog";
import { PROCESS_COMMAND_DIALOG_NAME } from '~/store/processes/process-command-actions';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { ProcessCommandDialogData } from '~/store/processes/process-command-actions';
import { DefaultCodeSnippet } from "~/components/default-code-snippet/default-code-snippet";
import { compose } from 'redux';

type CssRules = 'codeSnippet';

const styles: StyleRulesCallback<CssRules> = theme => ({
    codeSnippet: {
        marginLeft: theme.spacing.unit * 3,
        marginRight: theme.spacing.unit * 3,
    }
});

export const ProcessCommandDialog = compose(
    withDialog(PROCESS_COMMAND_DIALOG_NAME),
    withStyles(styles),
)(
    (props: WithDialogProps<ProcessCommandDialogData> & WithStyles<CssRules>) =>
        <Dialog
            open={props.open}
            maxWidth="md"
            onClose={props.closeDialog}
            style={{ alignSelf: 'stretch' }}>
            <DialogTitle>{`Command - ${props.data.processName}`}</DialogTitle>
            <DefaultCodeSnippet
                className={props.classes.codeSnippet}
                lines={[props.data.command]} />
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