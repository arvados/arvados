// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@material-ui/core/';
import { Button, StyleRulesCallback, WithStyles, withStyles, CircularProgress } from '@material-ui/core';
import { WithDialogProps } from '~/store/dialog/with-dialog';

type CssRules = "button" | "lastButton" | "formContainer" | "textField" | "dialog" | "dialogTitle" | "progressIndicator" | "dialogActions";

const styles: StyleRulesCallback<CssRules> = theme => ({
    button: {
        marginLeft: theme.spacing.unit
    },
    lastButton: {
        marginLeft: theme.spacing.unit,
        marginRight: "20px",
    },
    formContainer: {
        display: "flex",
        flexDirection: "column",
        marginTop: "20px",
    },
    dialogTitle: {
        paddingBottom: "0"
    },
    textField: {
        marginTop: "32px",
    },
    dialog: {
        minWidth: "600px",
        minHeight: "320px"
    },
    progressIndicator: {
        position: "absolute",
        minWidth: "20px",
    },
    dialogActions: {
        marginBottom: "24px"
    }
});

interface DialogProjectProps {
    cancelLabel?: string;
    dialogTitle: string;
    formFields: React.ComponentType<InjectedFormProps<any>>;
    submitLabel?: string;
}

export const FormDialog = withStyles(styles)((props: DialogProjectProps & WithDialogProps<{}> & InjectedFormProps<any> & WithStyles<CssRules>) =>
    <Dialog
        open={props.open}
        onClose={props.closeDialog}
        disableBackdropClick={props.submitting}
        disableEscapeKeyDown={props.submitting}>
        <div className={props.classes.dialog}>
            <form>
                <DialogTitle className={props.classes.dialogTitle}>
                    {props.dialogTitle}
                </DialogTitle>
                <DialogContent className={props.classes.formContainer}>
                    <props.formFields {...props} />
                </DialogContent>
                <DialogActions className={props.classes.dialogActions}>
                    <Button
                        onClick={props.closeDialog}
                        className={props.classes.button}
                        color="primary"
                        disabled={props.submitting}>
                        {props.cancelLabel || 'Cancel'}
                    </Button>
                    <Button
                        onClick={props.handleSubmit}
                        className={props.classes.lastButton}
                        color="primary"
                        disabled={props.invalid || props.submitting || props.pristine}
                        variant="contained">
                        {props.submitLabel || 'Submit'}
                        {props.submitting && <CircularProgress size={20} className={props.classes.progressIndicator} />}
                    </Button>
                </DialogActions>
            </form>
        </div>
    </Dialog>
);


