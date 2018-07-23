// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import TextField from '@material-ui/core/TextField';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import { Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

import { Validator } from '../../utils/dialog-validator';

type CssRules = "button" | "lastButton" | "dialogContent" | "textField" | "dialog" | "dialogTitle";

const styles: StyleRulesCallback<CssRules> = theme => ({
    button: {
        marginLeft: theme.spacing.unit
    },
    lastButton: {
        marginLeft: theme.spacing.unit,
        marginRight: "20px",
    },
    dialogContent: {
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
    }
});

interface ProjectCreateProps {
    open: boolean;
    handleClose: () => void;
    onSubmit: (data: { name: string, description: string }) => void;
}

interface DialogState {
    name: string;
    description: string;
    isNameValid: boolean;
    isDescriptionValid: boolean;
}

export const DialogProjectCreate = withStyles(styles)(
    class extends React.Component<ProjectCreateProps & WithStyles<CssRules>> {
        state: DialogState = {
            name: '',
            description: '',
            isNameValid: false,
            isDescriptionValid: true
        };

        render() {
            const { name, description } = this.state;
            const { classes, open, handleClose } = this.props;

            return (
                <Dialog
                    open={open}
                    onClose={handleClose}>
                    <div className={classes.dialog}>
                        <DialogTitle id="form-dialog-title" className={classes.dialogTitle}>Create a project</DialogTitle>
                        <DialogContent className={classes.dialogContent}>
                            <Validator
                                value={name}
                                onChange={e => this.isNameValid(e)}
                                isRequired={true}
                                render={hasError =>
                                    <TextField
                                        margin="dense"
                                        className={classes.textField}
                                        id="name"
                                        onChange={e => this.handleProjectName(e)}
                                        label="Project name"
                                        error={hasError}
                                        fullWidth/>}/>
                            <Validator
                                value={description}
                                onChange={e => this.isDescriptionValid(e)}
                                isRequired={false}
                                render={hasError =>
                                    <TextField
                                        margin="dense"
                                        className={classes.textField}
                                        id="description"
                                        onChange={e => this.handleDescriptionValue(e)}
                                        label="Description - optional"
                                        error={hasError}
                                        fullWidth/>}/>
                        </DialogContent>
                        <DialogActions>
                            <Button onClick={handleClose} className={classes.button} color="primary">CANCEL</Button>
                            <Button onClick={this.handleSubmit} className={classes.lastButton} color="primary"
                                    disabled={!this.state.isNameValid || (!this.state.isDescriptionValid && description.length > 0)}
                                    variant="raised">CREATE A PROJECT</Button>
                        </DialogActions>
                    </div>
                </Dialog>
            );
        }

        handleSubmit = () => {
            this.props.onSubmit({
                name: this.state.name,
                description: this.state.description
            });
        }

        handleProjectName(e: React.ChangeEvent<HTMLInputElement>) {
            this.setState({
                name: e.target.value,
            });
        }

        handleDescriptionValue(e: React.ChangeEvent<HTMLInputElement>) {
            this.setState({
                description: e.target.value,
            });
        }

        isNameValid(value: boolean | string) {
            this.setState({
                isNameValid: value,
            });
        }

        isDescriptionValid(value: boolean | string) {
            this.setState({
                isDescriptionValid: value,
            });
        }
    }
);
