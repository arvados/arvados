// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from "redux-form";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, DialogContentText, CircularProgress } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { TextField } from "../text-field/text-field";

export const RenameDialog = (props: WithDialogProps<string> & InjectedFormProps<{ name: string }>) =>
    <form>
        <Dialog open={props.open}>
            <DialogTitle>{`Rename`}</DialogTitle>
            <DialogContent>
                <DialogContentText>
                    {`Please, enter a new name for ${props.data}`}
                </DialogContentText>
                <Field
                    name='name'
                    component={TextField}
                />
            </DialogContent>
            <DialogActions>
                <Button
                    variant='text'
                    color='primary'
                    disabled={props.submitting}
                    onClick={props.closeDialog}>
                    Cancel
                    </Button>
                <Button
                    variant='contained'
                    color='primary'
                    type='submit'
                    onClick={props.handleSubmit}
                    disabled={props.pristine || props.invalid || props.submitting}>
                    {props.submitting
                        ? <CircularProgress size={20} />
                        : 'Ok'}
                </Button>
            </DialogActions>
        </Dialog>
    </form>;
