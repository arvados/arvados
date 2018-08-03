// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field } from 'redux-form';
import { compose } from 'redux';
import { ArvadosTheme } from '../../common/custom-theme';
import { Dialog, DialogActions, DialogContent, DialogTitle, TextField, StyleRulesCallback, withStyles, WithStyles, Button, CircularProgress } from '../../../node_modules/@material-ui/core';
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION } from '../../validators/create-project/create-project-validator';
import { COLLECTION_FORM_NAME } from '../../store/collections/updator/collection-updator-action';

type CssRules = 'content' | 'actions' | 'textField' | 'buttonWrapper' | 'saveButton' | 'circularProgress';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    content: {
        display: 'flex',
        flexDirection: 'column'
    },
    actions: {
        margin: 0,
        padding: `${theme.spacing.unit}px ${theme.spacing.unit * 3 - theme.spacing.unit / 2}px 
                ${theme.spacing.unit * 3}px ${theme.spacing.unit * 3}px`
    },
    textField: {
        marginBottom: theme.spacing.unit * 3
    },
    buttonWrapper: {
        position: 'relative'
    },
    saveButton: {
        boxShadow: 'none'
    },
    circularProgress: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        margin: 'auto'
    }
});

interface DialogCollectionDataProps {
    open: boolean;
    handleSubmit: any;
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

interface DialogCollectionAction {
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
}

type DialogCollectionProps = DialogCollectionDataProps & DialogCollectionAction & WithStyles<CssRules>;

interface TextFieldProps {
    label: string;
    floatinglabeltext: string;
    className?: string;
    input?: string;
    meta?: any;
}

export const DialogCollectionUpdate = compose(
    reduxForm({ form: COLLECTION_FORM_NAME }),
    withStyles(styles))(

        class DialogCollectionUpdate extends React.Component<DialogCollectionProps> {

            render() {
                const { classes, open, handleClose, handleSubmit, onSubmit, submitting, invalid, pristine } = this.props;
                return (
                    <Dialog open={open}
                        onClose={handleClose}
                        fullWidth={true}
                        maxWidth='sm'
                        disableBackdropClick={true}
                        disableEscapeKeyDown={true}>

                        <form onSubmit={handleSubmit((data: any) => onSubmit(data))}>
                            <DialogTitle>Edit Collection</DialogTitle>
                            <DialogContent className={classes.content}>
                                <Field name="name"
                                    disabled={submitting}
                                    component={this.renderTextField}
                                    floatinglabeltext="Collection Name"
                                    validate={COLLECTION_NAME_VALIDATION}
                                    className={classes.textField}
                                    label="Collection Name" />
                                <Field name="description"
                                    disabled={submitting}
                                    component={this.renderTextField}
                                    floatinglabeltext="Description - optional"
                                    validate={COLLECTION_DESCRIPTION_VALIDATION}
                                    className={classes.textField}
                                    label="Description - optional" />
                            </DialogContent>
                            <DialogActions className={classes.actions}>
                                <Button onClick={handleClose} color="primary"
                                    disabled={submitting}>CANCEL</Button>
                                <div className={classes.buttonWrapper}>
                                    <Button type="submit" className={classes.saveButton}
                                        color="primary"
                                        disabled={invalid || submitting || pristine}
                                        variant="contained">
                                        SAVE
                                    </Button>
                                    {submitting && <CircularProgress size={20} className={classes.circularProgress} />}
                                </div>
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