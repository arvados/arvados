// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { REPOSITORIES_SAMPLE_GIT_DIALOG } from "~/store/repositories/repositories-actions";
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from "redux";

type CssRules = 'codeSnippet' | 'link' | 'spacing';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: '1px solid',
        borderColor: theme.palette.grey["400"],
    },
    link: {
        textDecoration: 'none',
        color: theme.palette.primary.main,
        "&:hover": {
            color: theme.palette.primary.dark,
            transition: 'all 0.5s ease'
        }
    },
    spacing: {
        paddingTop: theme.spacing.unit * 2
    }
});

interface RepositoriesSampleGitDataProps {
    uuidPrefix: string;
}

type RepositoriesSampleGitProps = RepositoriesSampleGitDataProps & WithStyles<CssRules>;

export const RepositoriesSampleGitDialog = compose(
    withDialog(REPOSITORIES_SAMPLE_GIT_DIALOG),
    withStyles(styles))(
        (props: WithDialogProps<RepositoriesSampleGitProps> & RepositoriesSampleGitProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth='sm'>
                <DialogTitle>Sample git quick start:</DialogTitle>
                <DialogContent>
                    <DefaultCodeSnippet
                        className={props.classes.codeSnippet}
                        lines={[snippetText(props.data.uuidPrefix)]} />
                    <Typography variant='body1' className={props.classes.spacing}>
                        See also:
                        <div><a href="https://doc.arvados.org/user/getting_started/ssh-access-unix.html" className={props.classes.link} target="_blank">SSH access</a></div>
                        <div><a href="https://doc.arvados.org/user/tutorials/tutorial-firstscript.html" className={props.classes.link} target="_blank">Writing a Crunch Script</a></div>
                    </Typography>
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

const snippetText = (uuidPrefix: string) => `git clone git@git.${uuidPrefix}.arvadosapi.com:arvados.git
cd arvados
# edit files
git add the/files/you/changed
git commit
git push
`;
