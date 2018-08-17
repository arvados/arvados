// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, Field } from 'redux-form';
import { compose } from 'redux';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, WithStyles, withStyles, Dialog, DialogTitle, DialogContent, DialogActions, CircularProgress, Button } from '../../../node_modules/@material-ui/core';
import { TextField } from '~/components/text-field/text-field';
import { PROJECT_FORM_NAME } from '~/store/project/project-action';
import { PROJECT_NAME_VALIDATION, PROJECT_DESCRIPTION_VALIDATION } from '~/validators/validators';

type CssRules = 'content' | 'actions' | 'buttonWrapper' | 'saveButton' | 'circularProgress';

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

interface DialogProjectDataProps {
    open: boolean;
    handleSubmit: any;
    submitting: boolean;
    invalid: boolean;
    pristine: boolean;
}

interface DialogProjectActionProps {
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
}

type DialogProjectProps = DialogProjectDataProps & DialogProjectActionProps & WithStyles<CssRules>;

export const DialogProjectUpdate = compose(
    reduxForm({ form: PROJECT_FORM_NAME }),
    withStyles(styles))(

        class DialogProjectUpdate extends React.Component<DialogProjectProps> {
            render() {
                const { handleSubmit, handleClose, onSubmit, open, classes, submitting, invalid, pristine } = this.props;
                return <Dialog open={open}
                    onClose={handleClose}
                    fullWidth={true}
                    maxWidth='sm'
                    disableBackdropClick={true}
                    disableEscapeKeyDown={true}>
                    <form onSubmit={handleSubmit((data: any) => onSubmit(data))}>
                        <DialogTitle>Edit Collection</DialogTitle>
                        <DialogContent className={classes.content}>
                            <Field name='name' 
                                disabled={submitting}
                                component={TextField}
                                validate={PROJECT_NAME_VALIDATION}
                                label="Project Name" />
                            <Field name='description' 
                                disabled={submitting}
                                component={TextField} 
                                validate={PROJECT_DESCRIPTION_VALIDATION}
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
                </Dialog>;
            }
        }
    );
