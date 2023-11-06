// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@material-ui/core/';
import { Button, StyleRulesCallback, WithStyles, withStyles, CircularProgress } from '@material-ui/core';
import { WithDialogProps } from 'store/dialog/with-dialog';

type CssRules = "button" | "lastButton" | "form" | "formContainer" | "dialogTitle" | "progressIndicator" | "dialogActions";

const styles: StyleRulesCallback<CssRules> = theme => ({
    button: {
        marginLeft: theme.spacing.unit
    },
    lastButton: {
        marginLeft: theme.spacing.unit,
        marginRight: "0",
    },
    form: {
        display: 'flex',
        overflowY: 'auto',
        flexDirection: 'column',
        flex: '0 1 auto',
    },
    formContainer: {
        display: "flex",
        flexDirection: "column",
        paddingBottom: "0",
    },
    dialogTitle: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    progressIndicator: {
        position: "absolute",
        minWidth: "20px",
    },
    dialogActions: {
        marginBottom: theme.spacing.unit,
        marginRight: theme.spacing.unit * 3,
    }
});

interface DialogProjectDataProps {
    cancelLabel?: string;
    dialogTitle: string;
    formFields: React.ComponentType<InjectedFormProps<any> & WithDialogProps<any>>;
    submitLabel?: string;
    cancelCallback?: Function;
    enableWhenPristine?: boolean;
    doNotDisableCancel?: boolean;
}

type DialogProjectProps = DialogProjectDataProps & WithDialogProps<{}> & InjectedFormProps<any> & WithStyles<CssRules>;

export const FormDialog = withStyles(styles)((props: DialogProjectProps) =>
    <Dialog
        open={props.open}
        onClose={props.closeDialog}
        disableBackdropClick
        disableEscapeKeyDown={props.submitting}
        fullWidth
        maxWidth='md'>
        <form data-cy='form-dialog' className={props.classes.form}>
            <DialogTitle className={props.classes.dialogTitle}>
                {props.dialogTitle}
            </DialogTitle>
            <DialogContent className={props.classes.formContainer}>
                <props.formFields {...props} />
            </DialogContent>
            <DialogActions className={props.classes.dialogActions}>
                <Button
                    data-cy='form-cancel-btn'
                    onClick={() => {
                        props.closeDialog();

                        if (props.cancelCallback) {
                            props.cancelCallback();
                            props.reset();
                            props.initialize({});
                        }
                    }}
                    className={props.classes.button}
                    color="primary"
                    disabled={props.doNotDisableCancel ? false : props.submitting}>
                    {props.cancelLabel || 'Cancel'}
                </Button>
                <Button
                    data-cy='form-submit-btn'
                    type="submit"
                    onClick={props.handleSubmit}
                    className={props.classes.lastButton}
                    color="primary"
                    disabled={props.invalid || props.submitting || (props.pristine && !props.enableWhenPristine)}
                    variant="contained">
                    {props.submitLabel || 'Submit'}
                    {props.submitting && <CircularProgress size={20} className={props.classes.progressIndicator} />}
                </Button>
            </DialogActions>
        </form>
    </Dialog>
);
