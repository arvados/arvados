// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field } from 'redux-form';
import { compose } from 'redux';
import TextField from '@material-ui/core/TextField';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import { Button, StyleRulesCallback, WithStyles, withStyles, CircularProgress } from '@material-ui/core';

import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION } from '../../validators/create-project/create-project-validator';

type CssRules = "button" | "lastButton" | "formContainer" | "textField" | "createProgress" | "dialogActions";

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
    },
    textField: {
        marginBottom: theme.spacing.unit * 3
    },
    createProgress: {
        position: "absolute",
        minWidth: "20px",
        right: "110px"
    },
    dialogActions: {
        marginBottom: theme.spacing.unit * 3
    }
});
interface DialogCollectionCreateProps {
    open: boolean;
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
    handleSubmit: any;
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

interface TextFieldProps {
    label: string;
    floatinglabeltext: string;
    className?: string;
    input?: string;
    meta?: any;
}

export const DialogCollectionCreate = compose(
    reduxForm({ form: 'collectionCreateDialog' }),
    withStyles(styles))(
    class DialogCollectionCreate extends React.Component<DialogCollectionCreateProps & WithStyles<CssRules>> {
        render() {
            const { classes, open, handleClose, handleSubmit, onSubmit, submitting, invalid, pristine } = this.props;

            return (
                <Dialog
                    open={open}
                    onClose={handleClose}
                    fullWidth={true}
                    maxWidth='sm'
                    disableBackdropClick={true}
                    disableEscapeKeyDown={true}>
                    <form onSubmit={handleSubmit((data: any) => onSubmit(data))}>
                        <DialogTitle id="form-dialog-title">Create a collection</DialogTitle>
                        <DialogContent className={classes.formContainer}>
                            <Field name="name"
                                    disabled={submitting}
                                    component={this.renderTextField}
                                    floatinglabeltext="Collection Name"
                                    validate={COLLECTION_NAME_VALIDATION}
                                    className={classes.textField}
                                    label="Collection Name"/>
                            <Field name="description"
                                    disabled={submitting}
                                    component={this.renderTextField}
                                    floatinglabeltext="Description - optional"
                                    validate={COLLECTION_DESCRIPTION_VALIDATION}
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
                                CREATE A COLLECTION
                            </Button>
                            {submitting && <CircularProgress size={20} className={classes.createProgress}/>}
                        </DialogActions>
                    </form>
                </Dialog>
            );
        }

        renderTextField = ({ input, label, meta: { touched, error }, ...custom }: TextFieldProps) => (
            <TextField
                helperText={touched && error}
                label={label}
                className={this.props.classes.textField}
                error={touched && !!error}
                autoComplete='off'
                {...input}
                {...custom}
            />
        )
    }
);
