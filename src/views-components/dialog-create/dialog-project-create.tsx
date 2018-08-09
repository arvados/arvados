// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field } from 'redux-form';
import { compose } from 'redux';
import { TextField } from '../../components/text-field/text-field';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@material-ui/core/';
import { Button, StyleRulesCallback, WithStyles, withStyles, CircularProgress } from '@material-ui/core';

import { PROJECT_NAME_VALIDATION, PROJECT_DESCRIPTION_VALIDATION } from '../../validators/create-project/create-project-validator';

type CssRules = "button" | "lastButton" | "formContainer" | "textField" | "dialog" | "dialogTitle" | "createProgress" | "dialogActions";

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
    createProgress: {
        position: "absolute",
        minWidth: "20px",
        right: "95px"
    },
    dialogActions: {
        marginBottom: "24px"
    }
});
interface DialogProjectProps {
    open: boolean;
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
    handleSubmit: any;
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

export const PROJECT_CREATE_DIALOG = "projectCreateDialog";

export const DialogProjectCreate = compose(
    reduxForm({ form: PROJECT_CREATE_DIALOG }),
    withStyles(styles))(
    class DialogProjectCreate extends React.Component<DialogProjectProps & WithStyles<CssRules>> {
        render() {
            const { classes, open, handleClose, handleSubmit, onSubmit, submitting, invalid, pristine } = this.props;

            return (
                <Dialog
                    open={open}
                    onClose={handleClose}
                    disableBackdropClick={true}
                    disableEscapeKeyDown={true}>
                    <div className={classes.dialog}>
                        <form onSubmit={handleSubmit((data: any) => onSubmit(data))}>
                            <DialogTitle id="form-dialog-title" className={classes.dialogTitle}>Create a
                                project</DialogTitle>
                            <DialogContent className={classes.formContainer}>
                                <Field name="name"
                                       component={TextField}
                                       validate={PROJECT_NAME_VALIDATION}
                                       className={classes.textField}
                                       label="Project Name"/>
                                <Field name="description"
                                       component={TextField}
                                       validate={PROJECT_DESCRIPTION_VALIDATION}
                                       className={classes.textField}
                                       label="Description - optional"/>
                            </DialogContent>
                            <DialogActions className={classes.dialogActions}>
                                <Button onClick={handleClose} className={classes.button} color="primary"
                                        disabled={submitting}>CANCEL</Button>
                                <Button type="submit"
                                        className={classes.lastButton}
                                        color="primary"
                                        disabled={invalid || submitting || pristine}
                                        variant="contained">
                                    CREATE A PROJECT
                                </Button>
                                {submitting && <CircularProgress size={20} className={classes.createProgress}/>}
                            </DialogActions>
                        </form>
                    </div>
                </Dialog>
            );
        }
    }
);
