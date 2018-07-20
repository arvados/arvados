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

import { NAME, DESCRIPTION } from '../../validators/create-project/create-project-validator';
import { isUniqName } from '../../validators/is-uniq-name';

interface ProjectCreateProps {
    open: boolean;
    pending: boolean;
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
    handleSubmit: any;
}

interface TextFieldProps {
    label: string;
    floatinglabeltext: string;
    className?: string;
    input?: string;
    meta?: any;
}

class DialogProjectCreate extends React.Component<ProjectCreateProps & WithStyles<CssRules>> {
    /*componentWillReceiveProps(nextProps: ProjectCreateProps) {
        const { error } = nextProps;

        TODO: Validation for other errors
        if (this.props.error !== error && error && error.includes("UniqueViolation")) {
            this.setState({ isUniqName: error });
        }
}*/

    render() {
        const { classes, open, handleClose, pending, handleSubmit, onSubmit } = this.props;

        return (
            <Dialog
                open={open}
                onClose={handleClose}>
                <div className={classes.dialog}>
                    <form onSubmit={handleSubmit((data: any) => onSubmit(data))}>
                        <DialogTitle id="form-dialog-title" className={classes.dialogTitle}>Create a project</DialogTitle>
                        <DialogContent className={classes.formContainer}>
                            <Field name="name"
                                component={this.renderTextField}
                                floatinglabeltext="Project Name"
                                validate={NAME}
                                className={classes.textField}
                                label="Project Name" />
                            <Field name="description"
                                component={this.renderTextField}
                                floatinglabeltext="Description"
                                validate={DESCRIPTION}
                                className={classes.textField}
                                label="Description" />
                        </DialogContent>
                        <DialogActions>
                            <Button onClick={handleClose} className={classes.button} color="primary" disabled={pending}>CANCEL</Button>
                            <Button type="submit"
                                className={classes.lastButton}
                                color="primary"
                                disabled={pending}
                                variant="contained">
                                CREATE A PROJECT
                            </Button>
                            {pending && <CircularProgress size={20} className={classes.createProgress} />}
                        </DialogActions>
                    </form>
                </div>
            </Dialog>
        );
    }

    // TODO Make it separate file
    renderTextField = ({ input, label, meta: { touched, error }, ...custom }: TextFieldProps) => (
        <TextField
            helperText={touched && error ? error : void 0}
            label={label}
            className={this.props.classes.textField}
            error={touched && !!error} 
            autoComplete='off'
            {...input}
            {...custom}
        />
    )
}

type CssRules = "button" | "lastButton" | "formContainer" | "textField" | "dialog" | "dialogTitle" | "createProgress";

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
});

export default compose(
    reduxForm({ form: 'projectCreateDialog',/* asyncValidate: isUniqName, asyncBlurFields: ["name"] */}),
    withStyles(styles)
)(DialogProjectCreate);