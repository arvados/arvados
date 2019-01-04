// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from "redux";
import { API_CLIENT_AUTHORIZATION_HELP_DIALOG } from '~/store/api-client-authorizations/api-client-authorizations-actions';

type CssRules = 'codeSnippet';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: `1px solid ${theme.palette.grey["400"]}`,
        '& pre': {
            fontSize: '0.815rem'
        }
    }
});

interface HelpApiClientAuthorizationDataProps {
    apiHost: string;
    apiToken: string;
    email: string;
}

export const HelpApiClientAuthorizationDialog = compose(
    withDialog(API_CLIENT_AUTHORIZATION_HELP_DIALOG),
    withStyles(styles))(
        (props: WithDialogProps<HelpApiClientAuthorizationDataProps> & WithStyles<CssRules>) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth='md'>
                <DialogTitle>HELP:</DialogTitle>
                <DialogContent>
                    <DefaultCodeSnippet
                        className={props.classes.codeSnippet}
                        lines={[snippetText(props.data)]} />
                        {/* // lines={snippetText2(props.data)} /> */}
                </DialogContent>
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

const snippetText = (data: HelpApiClientAuthorizationDataProps) => `### Pasting the following lines at a shell prompt will allow Arvados SDKs
### to authenticate to your account, ${data.email}

read ARVADOS_API_TOKEN <<EOF
${data.apiToken}
EOF
export ARVADOS_API_TOKEN ARVADOS_API_HOST=${data.apiHost}
unset ARVADOS_API_HOST_INSECURE`;
