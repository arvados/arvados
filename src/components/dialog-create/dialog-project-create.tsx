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

interface ProjectCreateProps {
  open: boolean;
  handleClose: () => void;
}

const DialogProjectCreate: React.SFC<ProjectCreateProps & WithStyles<CssRules>> = ({ classes, open, handleClose }) => {
  return (
    <Dialog
      open={open}
      onClose={handleClose}>
      <div className={classes.dialog}>
        <DialogTitle id="form-dialog-title">Create a project</DialogTitle>
        <DialogContent className={classes.dialogContent}>
          <TextField
            margin="dense"
            className={classes.textField}
            id="name"
            label="Project name"
            fullWidth />
          <TextField
            margin="dense"
            id="description"
            label="Description - optional"
            fullWidth />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} className={classes.button} color="primary">CANCEL</Button>
          <Button onClick={handleClose} className={classes.lastButton} color="primary" variant="raised">CREATE A PROJECT</Button>
        </DialogActions>
      </div>
    </Dialog>
  );
};

type CssRules = "button" | "lastButton" | "dialogContent" | "textField" | "dialog";

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
  textField: {
    marginBottom: "32px",
  },
  dialog: {
    minWidth: "550px",
    minHeight: "320px"
  }
});

export default withStyles(styles)(DialogProjectCreate);