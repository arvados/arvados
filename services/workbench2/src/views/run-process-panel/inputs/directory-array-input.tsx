// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    isRequiredInput,
    DirectoryArrayCommandInputParameter,
    Directory,
    CWLType
} from 'models/workflow';
import { Field } from 'redux-form';
import { ERROR_MESSAGE } from 'validators/require';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button, Divider, WithStyles, Typography } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { connect, DispatchProp } from 'react-redux';
import { initProjectsTreePicker, getSelectedNodes, treePickerActions, getProjectsTreePickerIds, FileOperationLocation, getFileOperationLocation, fileOperationLocationToPickerId } from 'store/tree-picker/tree-picker-actions';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { createSelector, createStructuredSelector } from 'reselect';
import { ChipsInput } from 'components/chips-input/chips-input';
import { identity, values, noop } from 'lodash';
import { InputProps } from '@material-ui/core/Input';
import { TreePicker } from 'store/tree-picker/tree-picker';
import { RootState } from 'store/store';
import { Chips } from 'components/chips/chips';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import withStyles from '@material-ui/core/styles/withStyles';
import { CollectionResource } from 'models/collection';
import { PORTABLE_DATA_HASH_PATTERN, ResourceKind } from 'models/resource';
import { Dispatch } from 'redux';
import { CollectionDirectory, CollectionFileType } from 'models/collection-file';

const LOCATION_REGEX = new RegExp("^(?:keep:)?(" + PORTABLE_DATA_HASH_PATTERN + ")(/.*)?$");
export interface DirectoryArrayInputProps {
    input: DirectoryArrayCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

export const DirectoryArrayInput = ({ input }: DirectoryArrayInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={DirectoryArrayInputComponent as any}
        parse={parseDirectories}
        format={formatDirectories}
        validate={validationSelector(input)} />;

interface FormattedDirectory {
    name: string;
    portableDataHash: string;
    subpath: string;
}

const parseDirectories = (directories: FileOperationLocation[] | string) =>
    typeof directories === 'string'
        ? undefined
        : directories.map(parse);

const parse = (directory: FileOperationLocation): Directory => ({
    class: CWLType.DIRECTORY,
    basename: directory.name,
    location: `keep:${directory.pdh}${directory.subpath}`,
});

const formatDirectories = (directories: Directory[] = []): FormattedDirectory[] =>
    directories ? directories.map(format).filter((dir): dir is FormattedDirectory => Boolean(dir)) : [];

const format = ({ location = '', basename = '' }: Directory): FormattedDirectory | undefined => {
    const match = LOCATION_REGEX.exec(location);

    if (match) {
        return {
            portableDataHash: match[1],
            subpath: match[2],
            name: basename,
        };
    }
    return undefined;
};

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
    directories: FileOperationLocation[];
}

interface DirectoryArrayInputDataProps {
    treePickerState: TreePicker;
}

const treePickerSelector = (state: RootState) => state.treePicker;

const mapStateToProps = createStructuredSelector({
    treePickerState: treePickerSelector,
});

interface DirectoryArrayInputActionProps {
    initProjectsTreePicker: (pickerId: string) => void;
    selectTreePickerNode: (pickerId: string, id: string | string[]) => void;
    deselectTreePickerNode: (pickerId: string, id: string | string[]) => void;
    getFileOperationLocation: (item: ProjectsTreePickerItem) => Promise<FileOperationLocation | undefined>;
}

const mapDispatchToProps = (dispatch: Dispatch): DirectoryArrayInputActionProps => ({
    initProjectsTreePicker: (pickerId: string) => dispatch<any>(initProjectsTreePicker(pickerId)),
    selectTreePickerNode: (pickerId: string, id: string | string[]) =>
        dispatch<any>(treePickerActions.SELECT_TREE_PICKER_NODE({
            pickerId, id, cascade: false
        })),
    deselectTreePickerNode: (pickerId: string, id: string | string[]) =>
        dispatch<any>(treePickerActions.DESELECT_TREE_PICKER_NODE({
            pickerId, id, cascade: false
        })),
    getFileOperationLocation: (item: ProjectsTreePickerItem) => dispatch<any>(getFileOperationLocation(item)),
});

const DirectoryArrayInputComponent = connect(mapStateToProps, mapDispatchToProps)(
    class DirectoryArrayInputComponent extends React.Component<GenericInputProps & DirectoryArrayInputDataProps & DirectoryArrayInputActionProps & DispatchProp & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    }, DirectoryArrayInputComponentState> {
        state: DirectoryArrayInputComponentState = {
            open: false,
            directories: [],
        };

        directoryRefreshTimeout = -1;

        componentDidMount() {
            this.props.initProjectsTreePicker(this.props.commandInput.id);
        }

        render() {
            return <>
                <this.input />
                <this.dialog />
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
            this.props.input.onChange(this.state.directories);
        }

        setDirectoriesFromResources = async (directories: (CollectionResource | CollectionDirectory)[]) => {
            const locations = (await Promise.all(
                directories.map(directory => (this.props.getFileOperationLocation(directory)))
            )).filter((location): location is FileOperationLocation => (
                location !== undefined
            ));

            this.setDirectories(locations);
        }

        refreshDirectories = () => {
            clearTimeout(this.directoryRefreshTimeout);
            this.directoryRefreshTimeout = window.setTimeout(this.setDirectoriesFromTree);
        }

        setDirectoriesFromTree = () => {
            const nodes = getSelectedNodes<ProjectsTreePickerItem>(this.props.commandInput.id)(this.props.treePickerState);
            const initialDirectories: (CollectionResource | CollectionDirectory)[] = [];
            const directories = nodes
                .reduce((directories, { value }) =>
                    (('kind' in value && value.kind === ResourceKind.COLLECTION) ||
                    ('type' in value && value.type === CollectionFileType.DIRECTORY))
                        ? directories.concat(value)
                        : directories, initialDirectories);
            this.setDirectoriesFromResources(directories);
        }

        setDirectories = (locations: FileOperationLocation[]) => {
            const deletedDirectories = this.state.directories
                .reduce((deletedDirectories, directory) =>
                    locations.some(({ uuid, subpath }) => uuid === directory.uuid && subpath === directory.subpath)
                        ? deletedDirectories
                        : [...deletedDirectories, directory]
                    , [] as FileOperationLocation[]);

            this.setState({ directories: locations });

            const ids = values(getProjectsTreePickerIds(this.props.commandInput.id));
            ids.forEach(pickerId => {
                this.props.deselectTreePickerNode(
                    pickerId,
                    deletedDirectories.map(fileOperationLocationToPickerId)
                );
            });
        };

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

        dialogContentStyles: CustomStyleRulesCallback<DialogContentCssRules> = ({ spacing }) => ({
            root: {
                display: 'flex',
                flexDirection: 'column',
            },
            pickerWrapper: {
                display: 'flex',
                flexDirection: 'column',
                flexBasis: `${spacing(8)}vh`,
                flexShrink: 1,
                minHeight: 0,
            },
            tree: {
                flex: 3,
                overflow: 'auto',
            },
            divider: {
                margin: `${spacing(1)}px 0`,
            },
            chips: {
                flex: 1,
                overflow: 'auto',
                padding: `${spacing(1)}px 0`,
                overflowX: 'hidden',
            },
        });

        dialog = withStyles(this.dialogContentStyles)(
            ({ classes }: WithStyles<DialogContentCssRules>) =>
                <Dialog
                    open={this.state.open}
                    onClose={this.closeDialog}
                    fullWidth
                    maxWidth='md' >
                    <DialogTitle>Choose directories</DialogTitle>
                    <DialogContent className={classes.root}>
                        <div className={classes.pickerWrapper}>
                            <div className={classes.tree}>
                                <ProjectsTreePicker
                                    pickerId={this.props.commandInput.id}
                                    currentUuids={this.state.directories.map(dir => fileOperationLocationToPickerId(dir))}
                                    includeCollections
                                    includeDirectories
                                    showSelection
                                    cascadeSelection={false}
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
        );

    });

type DialogContentCssRules = 'root' | 'pickerWrapper' | 'tree' | 'divider' | 'chips';
