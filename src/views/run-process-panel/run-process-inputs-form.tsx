// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { reduxForm, InjectedFormProps } from 'redux-form';
import { CommandInputParameter, CWLType, IntCommandInputParameter, BooleanCommandInputParameter, FileCommandInputParameter, DirectoryCommandInputParameter, DirectoryArrayCommandInputParameter, FloatArrayCommandInputParameter, IntArrayCommandInputParameter } from '~/models/workflow';
import { IntInput } from '~/views/run-process-panel/inputs/int-input';
import { StringInput } from '~/views/run-process-panel/inputs/string-input';
import { StringCommandInputParameter, FloatCommandInputParameter, isPrimitiveOfType, WorkflowInputsData, EnumCommandInputParameter, isArrayOfType, StringArrayCommandInputParameter, FileArrayCommandInputParameter } from '../../models/workflow';
import { FloatInput } from '~/views/run-process-panel/inputs/float-input';
import { BooleanInput } from './inputs/boolean-input';
import { FileInput } from './inputs/file-input';
import { connect } from 'react-redux';
import { compose } from 'redux';
import { Grid, StyleRulesCallback, withStyles, WithStyles } from '@material-ui/core';
import { EnumInput } from './inputs/enum-input';
import { DirectoryInput } from './inputs/directory-input';
import { StringArrayInput } from './inputs/string-array-input';
import { createStructuredSelector, createSelector } from 'reselect';
import { FileArrayInput } from './inputs/file-array-input';
import { DirectoryArrayInput } from './inputs/directory-array-input';
import { FloatArrayInput } from './inputs/float-array-input';
import { IntArrayInput } from './inputs/int-array-input';

export const RUN_PROCESS_INPUTS_FORM = 'runProcessInputsForm';

export interface RunProcessInputFormProps {
    inputs: CommandInputParameter[];
}

const inputsSelector = (props: RunProcessInputFormProps) =>
    props.inputs;

const initialValuesSelector = createSelector(
    inputsSelector,
    inputs => inputs.reduce(
        (values, input) => ({ ...values, [input.id]: input.value || input.default }),
        {}));

const propsSelector = createStructuredSelector({
    initialValues: initialValuesSelector,
});

const mapStateToProps = (_: any, props: RunProcessInputFormProps) =>
    propsSelector(props);

export const RunProcessInputsForm = compose(
    connect(mapStateToProps),
    reduxForm<WorkflowInputsData, RunProcessInputFormProps>({
        form: RUN_PROCESS_INPUTS_FORM
    }))(
        (props: InjectedFormProps & RunProcessInputFormProps) =>
            <form>
                <Grid container spacing={32}>
                    {props.inputs.map(input =>
                        <InputItem input={input} key={input.id} />)}
                </Grid>
            </form>);

type CssRules = 'inputItem';

const styles: StyleRulesCallback<CssRules> = theme => ({
    inputItem: {
        marginBottom: theme.spacing.unit * 2,
    }
});

const InputItem = withStyles(styles)(
    (props: WithStyles<CssRules> & { input: CommandInputParameter }) =>
        <Grid item xs={12} md={6} className={props.classes.inputItem}>
            {getInputComponent(props.input)}
        </Grid>);

const getInputComponent = (input: CommandInputParameter) => {
    switch (true) {
        case isPrimitiveOfType(input, CWLType.BOOLEAN):
            return <BooleanInput input={input as BooleanCommandInputParameter} />;

        case isPrimitiveOfType(input, CWLType.INT):
        case isPrimitiveOfType(input, CWLType.LONG):
            return <IntInput input={input as IntCommandInputParameter} />;

        case isPrimitiveOfType(input, CWLType.FLOAT):
        case isPrimitiveOfType(input, CWLType.DOUBLE):
            return <FloatInput input={input as FloatCommandInputParameter} />;

        case isPrimitiveOfType(input, CWLType.STRING):
            return <StringInput input={input as StringCommandInputParameter} />;

        case isPrimitiveOfType(input, CWLType.FILE):
            return <FileInput input={input as FileCommandInputParameter} />;

        case isPrimitiveOfType(input, CWLType.DIRECTORY):
            return <DirectoryInput input={input as DirectoryCommandInputParameter} />;

        case typeof input.type === 'object' &&
            !(input.type instanceof Array) &&
            input.type.type === 'enum':
            return <EnumInput input={input as EnumCommandInputParameter} />;

        case isArrayOfType(input, CWLType.STRING):
            return <StringArrayInput input={input as StringArrayCommandInputParameter} />;

        case isArrayOfType(input, CWLType.INT):
        case isArrayOfType(input, CWLType.LONG):
            return <IntArrayInput input={input as IntArrayCommandInputParameter} />;

        case isArrayOfType(input, CWLType.FLOAT):
        case isArrayOfType(input, CWLType.DOUBLE):
            return <FloatArrayInput input={input as FloatArrayCommandInputParameter} />;

        case isArrayOfType(input, CWLType.FILE):
            return <FileArrayInput input={input as FileArrayCommandInputParameter} />;

        case isArrayOfType(input, CWLType.DIRECTORY):
            return <DirectoryArrayInput input={input as DirectoryArrayCommandInputParameter} />;

        default:
            return null;
    }
};
