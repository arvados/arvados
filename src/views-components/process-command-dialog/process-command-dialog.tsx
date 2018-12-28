// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogActions, Button, StyleRulesCallback, WithStyles, withStyles, Tooltip, IconButton, CardHeader } from '@material-ui/core';
import { withDialog } from "~/store/dialog/with-dialog";
import { PROCESS_COMMAND_DIALOG_NAME } from '~/store/processes/process-command-actions';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { ProcessCommandDialogData } from '~/store/processes/process-command-actions';
import { DefaultCodeSnippet } from "~/components/default-code-snippet/default-code-snippet";
import { compose } from 'redux';
import * as CopyToClipboard from "react-copy-to-clipboard";
import { CopyIcon } from '~/components/icon/icon';

type CssRules = 'codeSnippet' | 'copyToClipboard';

const styles: StyleRulesCallback<CssRules> = theme => ({
    codeSnippet: {
        marginLeft: theme.spacing.unit * 3,
        marginRight: theme.spacing.unit * 3,
    },
    copyToClipboard: {
        marginRight: theme.spacing.unit,
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
            <CardHeader
                title={`Command - ${props.data.processName}`}
                action={
                    <Tooltip title="Copy to clipboard">
                        <CopyToClipboard text={props.data.command}>
                            <IconButton className={props.classes.copyToClipboard}>
                                <CopyIcon />
                            </IconButton>
                        </CopyToClipboard>
                    </Tooltip>
                } />
            <DefaultCodeSnippet
                className={props.classes.codeSnippet}
                lines={[props.data.command]} />
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