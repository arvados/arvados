// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogActions, Button, CardHeader, DialogContent } from '@material-ui/core';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { withDialog } from "~/store/dialog/with-dialog";
import { PROCESS_INPUT_DIALOG_NAME } from '~/store/processes/process-input-actions';
import { RunProcessInputsForm } from "~/views/run-process-panel/run-process-inputs-form";

export const ProcessInputDialog = withDialog(PROCESS_INPUT_DIALOG_NAME)(
    (props: WithDialogProps<any>) =>
        <Dialog
            open={props.open}
            maxWidth={false}
            onClose={props.closeDialog}>
            <CardHeader
                title="Inputs - Pipeline template that generates a config file from a template" />
            <DialogContent>
                <RunProcessInputsForm inputs={getInputs(props.data.containerRequest)} />
            </DialogContent>
            <DialogActions>
                <Button
                    variant='text'
                    color='primary'
                    onClick={props.closeDialog}>
                    Close
                </Button>
            </DialogActions>
        </Dialog>
);

const getInputs = (data: any) =>
    data && data.mounts["/var/lib/cwl/workflow.json"] ? data.mounts["/var/lib/cwl/workflow.json"].content.$graph.find(
        (a: any) => a.id === '#main').inputs.map(
            (it: any) => (
                {
                    type: it.type,
                    id: it.id,
                    label: it.label,
                    value: data.mounts["/var/lib/cwl/cwl.input.json"].content[it.id],
                    disabled: true
                }
            )
        ) : [];
