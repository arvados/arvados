// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { compose } from 'redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { WithDialogProps, withDialog } from "store/dialog/with-dialog";
import { SSH_KEY_PUBLIC_KEY_DIALOG } from 'store/auth/auth-action-ssh';
import { ArvadosTheme } from 'common/custom-theme';
import { DefaultCodeSnippet } from 'components/default-code-snippet/default-code-snippet';

type CssRules = 'codeSnippet';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    codeSnippet: {
        borderRadius: theme.spacing(0.5),
        border: '1px solid',
        borderColor: theme.palette.grey["400"],
        '& pre': {
            wordWrap: 'break-word',
            whiteSpace: 'pre-wrap'
        }
    },
});

interface PublicKeyDialogDataProps {
    name: string;
    publicKey: string;
}

export const PublicKeyDialog = compose(
    withDialog(SSH_KEY_PUBLIC_KEY_DIALOG),
    withStyles(styles))(
        ({ open, closeDialog, data, classes }: WithDialogProps<PublicKeyDialogDataProps> & WithStyles<CssRules>) =>
            <Dialog open={open}
                onClose={closeDialog}
                fullWidth
                maxWidth='sm'>
                <DialogTitle>{data.name} - SSH Key</DialogTitle>
                <DialogContent>
                    {data && data.publicKey && <DefaultCodeSnippet
                        className={classes.codeSnippet}
                        lines={data.publicKey.split(' ')} />}
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
    );
