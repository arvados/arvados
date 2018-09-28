// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, InjectedFormProps } from 'redux-form';
import { WorkflowResource, CommandInputParameter, CWLType, IntCommandInputParameter } from '~/models/workflow';
import { IntInput } from '~/views/run-process-panel/inputs/int-input';
import { StringInput } from '~/views/run-process-panel/inputs/string-input';
import { StringCommandInputParameter, FloatCommandInputParameter } from '../../models/workflow';
import { FloatInput } from '~/views/run-process-panel/inputs/float-input';

const RUN_PROCESS_INPUTS_FORM = 'runProcessInputsForm';

export interface RunProcessInputFormProps {
    inputs: CommandInputParameter[];
}

export const RunProcessInputsForm = reduxForm<any, RunProcessInputFormProps>({
    form: RUN_PROCESS_INPUTS_FORM
})((props: InjectedFormProps & RunProcessInputFormProps) =>
    <form>
        {props.inputs.map(input => {
            switch (true) {
                case input.type === CWLType.INT:
                case input.type === CWLType.LONG:
                    return <IntInput key={input.id} input={input as IntCommandInputParameter} />;
                case input.type === CWLType.FLOAT:
                case input.type === CWLType.DOUBLE:
                    return <FloatInput key={input.id} input={input as FloatCommandInputParameter} />;
                case input.type === CWLType.STRING:
                    return <StringInput key={input.id} input={input as StringCommandInputParameter} />;
                default:
                    return null;
            }
        })}
    </form>);