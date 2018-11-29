// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { memoize } from 'lodash/fp';
import { Field } from 'redux-form';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@material-ui/core';
import {
    isRequiredInput,
    DirectoryCommandInputParameter,
    CWLType,
    Directory
} from '~/models/workflow';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from '~/views-components/projects-tree-picker/projects-tree-picker';
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';
import { TreeItem } from '~/components/tree/tree';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { CollectionResource } from '~/models/collection';
import { ResourceKind } from '~/models/resource';
import { ERROR_MESSAGE } from '~/validators/require';

export interface DirectoryInputProps {
    input: DirectoryCommandInputParameter;
}
export const DirectoryInput = ({ input }: DirectoryInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={DirectoryInputComponent}
        format={format}
        parse={parse}
        validate={getValidation(input)} />;

const format = (value?: Directory) => value ? value.basename : '';

const parse = (directory: CollectionResource): Directory => ({
    class: CWLType.DIRECTORY,
    location: `keep:${directory.portableDataHash}`,
    basename: directory.name,
});

const getValidation = memoize(
    (input: DirectoryCommandInputParameter) => ([
        isRequiredInput(input)
            ? (directory?: Directory) => directory ? undefined : ERROR_MESSAGE
            : () => undefined,
    ])
);

interface DirectoryInputComponentState {
    open: boolean;
    directory?: CollectionResource;
}

const DirectoryInputComponent = connect()(
    class FileInputComponent extends React.Component<GenericInputProps & DispatchProp, DirectoryInputComponentState> {
        state: DirectoryInputComponentState = {
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
            this.setState({ open: true });
        }

        closeDialog = () => {
            this.setState({ open: false });
        }

        submit = () => {
            this.closeDialog();
            this.props.input.onChange(this.state.directory);
        }

        setDirectory = (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => {
            if ('kind' in data && data.kind === ResourceKind.COLLECTION) {
                this.setState({ directory: data });
            } else {
                this.setState({ directory: undefined });
            }
        }

        renderInput() {
            return <GenericInput
                component={props =>
                    <Input
                        readOnly
                        fullWidth
                        value={props.input.value}
                        error={props.meta.touched && !!props.meta.error}
                        disabled={props.commandInput.disabled}
                        onClick={!this.props.commandInput.disabled ? this.openDialog : undefined}
                        onKeyPress={!this.props.commandInput.disabled ? this.openDialog : undefined} />}
                {...this.props} />;
        }

        renderDialog() {
            return <Dialog
                open={this.state.open}
                onClose={this.closeDialog}
                fullWidth
                maxWidth='md'>
                <DialogTitle>Choose a directory</DialogTitle>
                <DialogContent>
                    <ProjectsTreePicker
                        pickerId={this.props.commandInput.id}
                        includeCollections
                        toggleItemActive={this.setDirectory} />
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.closeDialog}>Cancel</Button>
                    <Button
                        disabled={!this.state.directory}
                        variant='contained'
                        color='primary'
                        onClick={this.submit}>Ok</Button>
                </DialogActions>
            </Dialog>;
        }

    });


