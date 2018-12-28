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
    data && data.mounts.varLibCwlWorkflowJson ? data.mounts.varLibCwlWorkflowJson.content.graph[1].inputs.map((it: any) => (
        { type: it.type, id: it.id, label: it.label, value: getInputValue(it.id, data.mounts.varLibCwlCwlInputJson.content), disabled: true }
    )) : [];

const getInputValue = (id: string, data: any) => {
    switch (id) {
        case "#main/example_flag":
            return data.exampleFlag;
        case "#main/example_directory":
            return data.exampleDirectory;
        case "#main/example_double":
            return data.exampleDouble;
        case "#main/example_file":
            return data.exampleFile;
        case "#main/example_float":
            return data.exampleFloat;
        case "#main/example_int":
            return data.exampleInt;
        case "#main/example_long":
            return data.exampleLong;
        case "#main/example_null":
            return data.exampleNull;
        case "#main/example_string":
            return data.exampleString;
        case "#main/enum_type":
            return data.enumType;
        case "#main/multiple_collections":
            return data.multipleCollections;
        case "#main/example_string_array":
            return data.exampleStringArray;
        case "#main/example_int_array":
            return data.exampleIntArray;
        case "#main/example_float_array":
            return data.exampleFloatArray;
        case "#main/multiple_files":
            return data.multipleFiles;
        case "#main/collection":
            return data.collection;
        case "#main/optional_file_missing_label":
            return data.optionalFileMissingLabel;
        case "#main/optional_file":
            return data.optionalFile;
        case "#main/single_file":
            return data.singleFile;
        default:
            return data.exampleString;
    }
};