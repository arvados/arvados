// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, InjectedFormProps } from 'redux-form';
import { WorkflowResource, CommandInputParameter, CWLType, IntCommandInputParameter, BooleanCommandInputParameter, FileCommandInputParameter } from '~/models/workflow';
import { IntInput } from '~/views/run-process-panel/inputs/int-input';
import { StringInput } from '~/views/run-process-panel/inputs/string-input';
import { StringCommandInputParameter, FloatCommandInputParameter, File } from '../../models/workflow';
import { FloatInput } from '~/views/run-process-panel/inputs/float-input';
import { BooleanInput } from './inputs/boolean-input';
import { FileInput } from './inputs/file-input';
import { connect } from 'react-redux';
import { compose } from 'redux';

const RUN_PROCESS_INPUTS_FORM = 'runProcessInputsForm';

export interface RunProcessInputFormProps {
    inputs: CommandInputParameter[];
}

export const RunProcessInputsForm = compose(
    connect((_: any, props: RunProcessInputFormProps) => ({
        initialValues: props.inputs.reduce(
            (values, input) => ({ ...values, [input.id]: input.default }),
            {}),
    })),
    reduxForm<any, RunProcessInputFormProps>({
        form: RUN_PROCESS_INPUTS_FORM
    }))((props: InjectedFormProps & RunProcessInputFormProps) =>
        <form>
            {props.inputs.map(input => {
                switch (true) {
                    case input.type === CWLType.BOOLEAN:
                        return <BooleanInput key={input.id} input={input as BooleanCommandInputParameter} />;

                    case input.type === CWLType.INT:
                    case input.type === CWLType.LONG:
                        return <IntInput key={input.id} input={input as IntCommandInputParameter} />;

                    case input.type === CWLType.FLOAT:
                    case input.type === CWLType.DOUBLE:
                        return <FloatInput key={input.id} input={input as FloatCommandInputParameter} />;

                    case input.type === CWLType.STRING:
                        return <StringInput key={input.id} input={input as StringCommandInputParameter} />;

                    case input.type === CWLType.FILE:
                        return <FileInput key={input.id} input={input as FileCommandInputParameter} />;

                    default:
                        return null;
                }
            })}
        </form>);
