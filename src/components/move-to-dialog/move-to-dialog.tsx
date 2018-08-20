// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Field, InjectedFormProps, WrappedFieldProps } from "redux-form";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, CircularProgress } from "@material-ui/core";

import { WithDialogProps } from "~/store/dialog/with-dialog";
import { ProjectTreePicker } from "~/views-components/project-tree-picker/project-tree-picker";
import { MOVE_TO_VALIDATION } from "~/validators/validators";

export const MoveToDialog = (props: WithDialogProps<string> & InjectedFormProps<{ name: string }>) =>
    <form>
        <Dialog open={props.open}
            disableBackdropClick={true}
            disableEscapeKeyDown={true}>
            <DialogTitle>Move to</DialogTitle>
            <DialogContent>
                <Field
                    name="projectUuid"
                    component={Picker}
                    validate={MOVE_TO_VALIDATION} />
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
                    {props.submitting ? <CircularProgress size={20} /> : 'Move'}
                </Button>
            </DialogActions>
        </Dialog>
    </form>;

const Picker = (props: WrappedFieldProps) =>
    <div style={{ width: '400px', height: '144px', display: 'flex', flexDirection: 'column' }}>
       <ProjectTreePicker onChange={projectUuid => props.input.onChange(projectUuid)} /> 
    </div>;