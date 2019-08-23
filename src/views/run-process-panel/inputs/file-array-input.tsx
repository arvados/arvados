// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    isRequiredInput,
    FileArrayCommandInputParameter,
    File,
    CWLType
} from '~/models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from '~/validators/require';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button, Divider, WithStyles, Typography } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from '~/views-components/projects-tree-picker/projects-tree-picker';
import { connect, DispatchProp } from 'react-redux';
import { initProjectsTreePicker, getSelectedNodes, treePickerActions, getProjectsTreePickerIds } from '~/store/tree-picker/tree-picker-actions';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { CollectionFile, CollectionFileType } from '~/models/collection-file';
import { createSelector, createStructuredSelector } from 'reselect';
import { ChipsInput } from '~/components/chips-input/chips-input';
import { identity, values, noop } from 'lodash';
import { InputProps } from '@material-ui/core/Input';
import { TreePicker } from '~/store/tree-picker/tree-picker';
import { RootState } from '~/store/store';
import { Chips } from '~/components/chips/chips';
import withStyles, { StyleRulesCallback } from '@material-ui/core/styles/withStyles';

export interface FileArrayInputProps {
    input: FileArrayCommandInputParameter;
}
export const FileArrayInput = ({ input }: FileArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={FileArrayInputComponent}
        parse={parseFiles}
        format={formatFiles}
        validate={validationSelector(input)} />;

const parseFiles = (files: CollectionFile[] | string) =>
    typeof files === 'string'
        ? undefined
        : files.map(parse);

const parse = (file: CollectionFile): File => ({
    class: CWLType.FILE,
    basename: file.name,
    location: `keep:${file.id}`,
    path: file.path,
});

const formatFiles = (files: File[] = []) =>
    files.map(format);

const format = (file: File): CollectionFile => ({
    id: file.location
        ? file.location.replace('keep:', '')
        : '',
    name: file.basename || '',
    path: file.path || '',
    size: 0,
    type: CollectionFileType.FILE,
    url: '',
});

const validationSelector = createSelector(
    isRequiredInput,
    isRequired => isRequired
        ? [required]
        : undefined
);

const required = (value?: File[]) =>
    value && value.length > 0
        ? undefined
        : ERROR_MESSAGE;
interface FileArrayInputComponentState {
    open: boolean;
    files: CollectionFile[];
}

interface FileArrayInputComponentProps {
    treePickerState: TreePicker;
}

const treePickerSelector = (state: RootState) => state.treePicker;

const mapStateToProps = createStructuredSelector({
    treePickerState: treePickerSelector,
});

const FileArrayInputComponent = connect(mapStateToProps)(
    class FileArrayInputComponent extends React.Component<FileArrayInputComponentProps & GenericInputProps & DispatchProp, FileArrayInputComponentState> {
        state: FileArrayInputComponentState = {
            open: false,
            files: [],
        };

        fileRefreshTimeout = -1;

        componentDidMount() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));
        }

        render() {
            return <>
                <this.input />
                <this.dialog />
            </>;
        }

        openDialog = () => {
            this.setFilesFromProps(this.props.input.value);
            this.setState({ open: true });
        }

        closeDialog = () => {
            this.setState({ open: false });
        }

        submit = () => {
            this.closeDialog();
            this.props.input.onChange(this.state.files);
        }

        setFiles = (files: CollectionFile[]) => {

            const deletedFiles = this.state.files
                .reduce((deletedFiles, file) =>
                    files.some(({ id }) => id === file.id)
                        ? deletedFiles
                        : [...deletedFiles, file]
                    , []);

            this.setState({ files });

            const ids = values(getProjectsTreePickerIds(this.props.commandInput.id));
            ids.forEach(pickerId => {
                this.props.dispatch(
                    treePickerActions.DESELECT_TREE_PICKER_NODE({
                        pickerId, id: deletedFiles.map(({ id }) => id),
                    })
                );
            });

        }

        setFilesFromProps = (files: CollectionFile[]) => {

            const addedFiles = files
                .reduce((addedFiles, file) =>
                    this.state.files.some(({ id }) => id === file.id)
                        ? addedFiles
                        : [...addedFiles, file]
                    , []);

            const ids = values(getProjectsTreePickerIds(this.props.commandInput.id));
            ids.forEach(pickerId => {
                this.props.dispatch(
                    treePickerActions.SELECT_TREE_PICKER_NODE({
                        pickerId, id: addedFiles.map(({ id }) => id),
                    })
                );
            });

            this.setFiles(files);

        }

        refreshFiles = () => {
            clearTimeout(this.fileRefreshTimeout);
            this.fileRefreshTimeout = setTimeout(this.setSelectedFiles);
        }

        setSelectedFiles = () => {
            const nodes = getSelectedNodes<ProjectsTreePickerItem>(this.props.commandInput.id)(this.props.treePickerState);
            const initialFiles: CollectionFile[] = [];
            const files = nodes
                .reduce((files, { value }) =>
                    'type' in value && value.type === CollectionFileType.FILE
                        ? files.concat(value)
                        : files, initialFiles);

            this.setFiles(files);
        }
        input = () =>
            <GenericInput
                component={this.chipsInput}
                {...this.props} />

        chipsInput = () =>
            <ChipsInput
                value={this.props.input.value}
                disabled={this.props.commandInput.disabled}
                onChange={noop}
                createNewValue={identity}
                getLabel={(file: CollectionFile) => file.name}
                inputComponent={this.textInput} />

        textInput = (props: InputProps) =>
            <Input
                {...props}
                error={this.props.meta.touched && !!this.props.meta.error}
                readOnly
                disabled={this.props.commandInput.disabled}
                onClick={!this.props.commandInput.disabled ? this.openDialog : undefined}
                onKeyPress={!this.props.commandInput.disabled ? this.openDialog : undefined}
                onBlur={this.props.input.onBlur} />

        dialog = () =>
            <Dialog
                open={this.state.open}
                onClose={this.closeDialog}
                fullWidth
                maxWidth='md' >
                <DialogTitle>Choose files</DialogTitle>
                <DialogContent>
                    <this.dialogContent />
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.closeDialog}>Cancel</Button>
                    <Button
                        variant='contained'
                        color='primary'
                        onClick={this.submit}>Ok</Button>
                </DialogActions>
            </Dialog>

        dialogContentStyles: StyleRulesCallback<DialogContentCssRules> = ({ spacing }) => ({
            root: {
                display: 'flex',
                flexDirection: 'column',
                height: `${spacing.unit * 8}vh`,
            },
            tree: {
                flex: 3,
                overflow: 'auto',
            },
            divider: {
                margin: `${spacing.unit}px 0`,
            },
            chips: {
                flex: 1,
                overflow: 'auto',
                padding: `${spacing.unit}px 0`,
                overflowX: 'hidden',
            },
        })

        dialogContent = withStyles(this.dialogContentStyles)(
            ({ classes }: WithStyles<DialogContentCssRules>) =>
                <div className={classes.root}>
                    <div className={classes.tree}>
                        <ProjectsTreePicker
                            pickerId={this.props.commandInput.id}
                            includeCollections
                            includeFiles
                            showSelection
                            toggleItemSelection={this.refreshFiles} />
                    </div>
                    <Divider />
                    <div className={classes.chips}>
                        <Typography variant='subtitle1'>Selected files ({this.state.files.length}):</Typography>
                        <Chips
                            orderable
                            deletable
                            values={this.state.files}
                            onChange={this.setFiles}
                            getLabel={(file: CollectionFile) => file.name} />
                    </div>
                </div>
        );

    });

type DialogContentCssRules = 'root' | 'tree' | 'divider' | 'chips';



