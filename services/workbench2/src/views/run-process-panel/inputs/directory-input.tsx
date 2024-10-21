// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { memoize } from 'lodash/fp';
import { Field } from 'redux-form';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import {
    isRequiredInput,
    DirectoryCommandInputParameter,
    CWLType,
    Directory
} from 'models/workflow';
import { GenericInputProps, GenericInput } from './generic-input';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { FileOperationLocation, getFileOperationLocation, initProjectsTreePicker } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { ERROR_MESSAGE } from 'validators/require';
import { Dispatch } from 'redux';

export interface DirectoryInputProps {
    input: DirectoryCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

type DialogContentCssRules = 'root' | 'pickerWrapper';

export const DirectoryInput = ({ input, options }: DirectoryInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={DirectoryInputComponent as any}
        format={format}
        parse={parse}
        {...{
            options
        }}
        validate={getValidation(input)} />;

const format = (value?: Directory) => value ? value.basename : '';

const parse = (directory: FileOperationLocation): Directory => ({
    class: CWLType.DIRECTORY,
    location: `keep:${directory.pdh}${directory.subpath}`,
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
    directory?: FileOperationLocation;
}

interface DirectoryInputActionProps {
    initProjectsTreePicker: (pickerId: string) => void;
    getFileOperationLocation: (item: ProjectsTreePickerItem) => Promise<FileOperationLocation | undefined>;
}

const mapDispatchToProps = (dispatch: Dispatch): DirectoryInputActionProps => ({
    initProjectsTreePicker: (pickerId: string) => dispatch<any>(initProjectsTreePicker(pickerId)),
    getFileOperationLocation: (item: ProjectsTreePickerItem) => dispatch<any>(getFileOperationLocation(item)),
});

const DirectoryInputComponent = connect(null, mapDispatchToProps)(
    class FileInputComponent extends React.Component<GenericInputProps & DirectoryInputActionProps & DispatchProp & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    }, DirectoryInputComponentState> {
        state: DirectoryInputComponentState = {
            open: false,
        };

        componentDidMount() {
            this.props.initProjectsTreePicker(this.props.commandInput.id);
        }

        render() {
            return <>
                {this.renderInput()}
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
            this.props.input.onChange(this.state.directory);
        }

        setDirectory = async (_: {}, { data: item }: TreeItem<ProjectsTreePickerItem>) => {
            const location = await this.props.getFileOperationLocation(item);
            this.setState({ directory: location });
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

        dialogContentStyles: CustomStyleRulesCallback<DialogContentCssRules> = ({ spacing }) => ({
            root: {
                display: 'flex',
                flexDirection: 'column',
                height: "80vh",
            },
            pickerWrapper: {
                display: 'flex',
                flexDirection: 'column',
                height: "100%",
            },
        });

        dialog = withStyles(this.dialogContentStyles)(
            ({ classes }: WithStyles<DialogContentCssRules>) =>
                <Dialog
                    open={this.state.open}
                    onClose={this.closeDialog}
                    fullWidth
                    data-cy="choose-a-directory-dialog"
                    maxWidth='md'>
                    <DialogTitle>Choose a directory</DialogTitle>
                    <DialogContent className={classes.root}>
                        <div className={classes.pickerWrapper}>
                            <ProjectsTreePicker
                                pickerId={this.props.commandInput.id}
                                includeCollections
                                includeDirectories
                                cascadeSelection={false}
                                options={this.props.options}
                                toggleItemActive={this.setDirectory} />
                        </div>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={this.closeDialog}>Cancel</Button>
                        <Button
                            disabled={!this.state.directory}
                            variant='contained'
                            color='primary'
                            onClick={this.submit}>Ok</Button>
                    </DialogActions>
                </Dialog>
        );

    });
