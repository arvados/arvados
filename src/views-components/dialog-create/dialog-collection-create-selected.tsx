// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from "redux-form";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, CircularProgress } from "@material-ui/core";
import { WithDialogProps } from "../../store/dialog/with-dialog";
import { TextField } from "../../components/text-field/text-field";
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION } from "../../validators/create-project/create-project-validator";

export const DialogCollectionCreateWithSelected = (props: WithDialogProps<string> & InjectedFormProps<{ name: string }>) =>
    <form>
        <Dialog open={props.open}
            disableBackdropClick={true}
            disableEscapeKeyDown={true}>
            <DialogTitle>Create a collection</DialogTitle>
            <DialogContent style={{ display: 'flex' }}>
                <div>
                    <Field
                        name='name'
                        component={TextField}
                        validate={COLLECTION_NAME_VALIDATION}
                        label="Collection Name" />
                    <Field
                        name='description'
                        component={TextField}
                        validate={COLLECTION_DESCRIPTION_VALIDATION}
                        label="Description - optional" />
                </div>
                <div style={{ overflowY: 'auto' }}>TREE</div>
            </DialogContent>
            <DialogActions>
                <Button
                    variant='flat'
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
                        : 'Create a collection'}
                </Button>
            </DialogActions>
        </Dialog>
    </form>;