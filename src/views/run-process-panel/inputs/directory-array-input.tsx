// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    isRequiredInput,
    DirectoryArrayCommandInputParameter,
    Directory,
    CWLType
} from '~/models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from '~/validators/require';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button, Divider, WithStyles, Typography } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from '~/views-components/projects-tree-picker/projects-tree-picker';
import { connect, DispatchProp } from 'react-redux';
import { initProjectsTreePicker, getSelectedNodes, treePickerActions, getProjectsTreePickerIds, getAllNodes } from '~/store/tree-picker/tree-picker-actions';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { createSelector, createStructuredSelector } from 'reselect';
import { ChipsInput } from '~/components/chips-input/chips-input';
import { identity, values, noop } from 'lodash';
import { InputProps } from '@material-ui/core/Input';
import { TreePicker } from '~/store/tree-picker/tree-picker';
import { RootState } from '~/store/store';
import { Chips } from '~/components/chips/chips';
import withStyles, { StyleRulesCallback } from '@material-ui/core/styles/withStyles';
import { CollectionResource } from '~/models/collection';
import { ResourceKind } from '~/models/resource';

export interface DirectoryArrayInputProps {
    input: DirectoryArrayCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

export const DirectoryArrayInput = ({ input }: DirectoryArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={DirectoryArrayInputComponent}
        parse={parseDirectories}
        format={formatDirectories}
        validate={validationSelector(input)} />;

interface FormattedDirectory {
    name: string;
    portableDataHash: string;
}

const parseDirectories = (directories: CollectionResource[] | string) =>
    typeof directories === 'string'
        ? undefined
        : directories.map(parse);

const parse = (directory: CollectionResource): Directory => ({
    class: CWLType.DIRECTORY,
    basename: directory.name,
    location: `keep:${directory.portableDataHash}`,
});

const formatDirectories = (directories: Directory[] = []) =>
    directories ? directories.map(format) : [];

const format = ({ location = '', basename = '' }: Directory): FormattedDirectory => ({
    portableDataHash: location.replace('keep:', ''),
    name: basename,
});

const validationSelector = createSelector(
    isRequiredInput,
    isRequired => isRequired
        ? [required]
        : undefined
);

const required = (value?: Directory[]) =>
    value && value.length > 0
        ? undefined
        : ERROR_MESSAGE;
interface DirectoryArrayInputComponentState {
    open: boolean;
    directories: CollectionResource[];
    prevDirectories: CollectionResource[];
}

interface DirectoryArrayInputComponentProps {
    treePickerState: TreePicker;
}

const treePickerSelector = (state: RootState) => state.treePicker;

const mapStateToProps = createStructuredSelector({
    treePickerState: treePickerSelector,
});

const DirectoryArrayInputComponent = connect(mapStateToProps)(
    class DirectoryArrayInputComponent extends React.Component<DirectoryArrayInputComponentProps & GenericInputProps & DispatchProp & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    }, DirectoryArrayInputComponentState> {
        state: DirectoryArrayInputComponentState = {
            open: false,
            directories: [],
            prevDirectories: [],
        };

        directoryRefreshTimeout = -1;

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
            this.setDirectoriesFromProps(this.props.input.value);
            this.setState({ open: true });
        }

        closeDialog = () => {
            this.setState({ open: false });
        }

        submit = () => {
            this.closeDialog();
            this.props.input.onChange(this.state.directories);
        }

        setDirectories = (directories: CollectionResource[]) => {

            const deletedDirectories = this.state.directories
                .reduce((deletedDirectories, directory) =>
                    directories.some(({ uuid }) => uuid === directory.uuid)
                        ? deletedDirectories
                        : [...deletedDirectories, directory]
                    , []);

            this.setState({ directories });

            const ids = values(getProjectsTreePickerIds(this.props.commandInput.id));
            ids.forEach(pickerId => {
                this.props.dispatch(
                    treePickerActions.DESELECT_TREE_PICKER_NODE({
                        pickerId, id: deletedDirectories.map(({ uuid }) => uuid),
                    })
                );
            });

        }

        setDirectoriesFromProps = (formattedDirectories: FormattedDirectory[]) => {
            const nodes = getAllNodes<ProjectsTreePickerItem>(this.props.commandInput.id)(this.props.treePickerState);
            const initialDirectories: CollectionResource[] = [];
            const directories = nodes
                .reduce((directories, { value }) =>
                    'kind' in value &&
                        value.kind === ResourceKind.COLLECTION &&
                        formattedDirectories.find(({ portableDataHash, name }) => value.portableDataHash === portableDataHash && value.name === name)
                        ? directories.concat(value)
                        : directories, initialDirectories);

            const addedDirectories = directories
                .reduce((addedDirectories, directory) =>
                    this.state.directories.find(({ uuid }) =>
                        uuid === directory.uuid)
                        ? addedDirectories
                        : [...addedDirectories, directory]
                    , []);

            const ids = values(getProjectsTreePickerIds(this.props.commandInput.id));
            ids.forEach(pickerId => {
                this.props.dispatch(
                    treePickerActions.SELECT_TREE_PICKER_NODE({
                        pickerId, id: addedDirectories.map(({ uuid }) => uuid),
                    })
                );
            });

            const orderedDirectories = formattedDirectories.reduce((dirs, formattedDir) => {
                const dir = directories.find(({ portableDataHash, name }) => portableDataHash === formattedDir.portableDataHash && name === formattedDir.name);
                return dir
                    ? [...dirs, dir]
                    : dirs;
            }, []);

            this.setDirectories(orderedDirectories);

        }

        refreshDirectories = () => {
            clearTimeout(this.directoryRefreshTimeout);
            this.directoryRefreshTimeout = setTimeout(this.setSelectedFiles);
        }

        setSelectedFiles = () => {
            const nodes = getSelectedNodes<ProjectsTreePickerItem>(this.props.commandInput.id)(this.props.treePickerState);
            const initialDirectories: CollectionResource[] = [];
            const directories = nodes
                .reduce((directories, { value }) =>
                    'kind' in value && value.kind === ResourceKind.COLLECTION
                        ? directories.concat(value)
                        : directories, initialDirectories);
            this.setDirectories(directories);
        }
        input = () =>
            <GenericInput
                component={this.chipsInput}
                {...this.props} />

        chipsInput = () =>
            <ChipsInput
                values={this.props.input.value}
                onChange={noop}
                disabled={this.props.commandInput.disabled}
                createNewValue={identity}
                getLabel={(data: FormattedDirectory) => data.name}
                inputComponent={this.textInput} />

        textInput = (props: InputProps) =>
            <Input
                {...props}
                error={this.props.meta.touched && !!this.props.meta.error}
                readOnly
                onClick={!this.props.commandInput.disabled ? this.openDialog : undefined}
                onKeyPress={!this.props.commandInput.disabled ? this.openDialog : undefined}
                onBlur={this.props.input.onBlur}
                disabled={this.props.commandInput.disabled} />

        dialog = () =>
            <Dialog
                open={this.state.open}
                onClose={this.closeDialog}
                fullWidth
                maxWidth='md' >
                <DialogTitle>Choose collections</DialogTitle>
                <DialogContent>
                    <this.dialogContent />
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.closeDialog}>Cancel</Button>
                    <Button
                        data-cy='ok-button'
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
                            showSelection
                            options={this.props.options}
                            toggleItemSelection={this.refreshDirectories} />
                    </div>
                    <Divider />
                    <div className={classes.chips}>
                        <Typography variant='subtitle1'>Selected collections ({this.state.directories.length}):</Typography>
                        <Chips
                            orderable
                            deletable
                            values={this.state.directories}
                            onChange={this.setDirectories}
                            getLabel={(directory: CollectionResource) => directory.name} />
                    </div>
                </div>
        );

    });

type DialogContentCssRules = 'root' | 'tree' | 'divider' | 'chips';
