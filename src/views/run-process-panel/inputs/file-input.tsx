// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { memoize } from 'lodash/fp';
import {
    isRequiredInput,
    FileCommandInputParameter,
    File,
    CWLType
} from 'models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from 'validators/require';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { connect, DispatchProp } from 'react-redux';
import { initProjectsTreePicker } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { CollectionFile, CollectionFileType } from 'models/collection-file';

export interface FileInputProps {
    input: FileCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}
export const FileInput = ({ input, options }: FileInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={FileInputComponent as any}
        format={format}
        parse={parse}
        {...{
            options
        }}
        validate={getValidation(input)} />;

const format = (value?: File) => value ? value.basename : '';

const parse = (file: CollectionFile): File => ({
    class: CWLType.FILE,
    location: `keep:${file.id}`,
    basename: file.name,
});

const getValidation = memoize(
    (input: FileCommandInputParameter) => ([
        isRequiredInput(input)
            ? (file?: File) => file ? undefined : ERROR_MESSAGE
            : () => undefined,
    ]));

interface FileInputComponentState {
    open: boolean;
    file?: CollectionFile;
}

const FileInputComponent = connect()(
    class FileInputComponent extends React.Component<GenericInputProps & DispatchProp & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    }, FileInputComponentState> {
        state: FileInputComponentState = {
            open: false,
        };

        componentDidMount() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));
        }

        render() {
            return <>
                {this.renderInput()}
                {this.renderDialog()}
            </>;
        }

        openDialog = () => {
            this.componentDidMount();
            this.setState({ open: true });
        }

        closeDialog = () => {
            this.setState({ open: false });
        }

        submit = () => {
            this.closeDialog();
            this.props.input.onChange(this.state.file);
        }

        setFile = (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => {
            if ('type' in data && data.type === CollectionFileType.FILE) {
                this.setState({ file: data });
            } else {
                this.setState({ file: undefined });
            }
        }

        renderInput() {
            return <GenericInput
                component={props =>
                    <Input
                        readOnly
                        fullWidth
                        disabled={props.commandInput.disabled}
                        value={props.input.value}
                        error={props.meta.touched && !!props.meta.error}
                        onClick={!props.commandInput.disabled ? this.openDialog : undefined}
                        onKeyPress={!props.commandInput.disabled ? this.openDialog : undefined} />}
                {...this.props} />;
        }

        renderDialog() {
            return <Dialog
                open={this.state.open}
                onClose={this.closeDialog}
                fullWidth
                data-cy="choose-a-file-dialog"
                maxWidth='md'>
                <DialogTitle>Choose a file</DialogTitle>
                <DialogContent>
                    <ProjectsTreePicker
                        pickerId={this.props.commandInput.id}
                        includeCollections
                        includeFiles
                        options={this.props.options}
                        toggleItemActive={this.setFile} />
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.closeDialog}>Cancel</Button>
                    <Button
                        disabled={!this.state.file}
                        variant='contained'
                        color='primary'
                        onClick={this.submit}>Ok</Button>
                </DialogActions>
            </Dialog>;
        }

    });


