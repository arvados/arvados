// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { Field } from 'redux-form';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@material-ui/core';
import {
    GenericCommandInputParameter
} from 'models/workflow';
import { GenericInput, GenericInputProps } from './generic-input';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { initProjectsTreePicker } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'views-components/projects-tree-picker/generic-projects-tree-picker';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { RootState } from 'store/store';
import { getUserUuid } from 'common/getuser';

export type ProjectCommandInputParameter = GenericCommandInputParameter<ProjectResource, ProjectResource>;

const require: any = (value?: ProjectResource) => (value === undefined);

export interface ProjectInputProps {
    input: ProjectCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}
export const ProjectInput = ({ input, options }: ProjectInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={ProjectInputComponent as any}
        format={format}
        validate={require}
        {...{
            options
        }} />;

const format = (value?: ProjectResource) => value ? value.name : '';

interface ProjectInputComponentState {
    open: boolean;
    project?: ProjectResource;
}

interface HasUserUuid {
    userUuid: string;
};

const mapStateToProps = (state: RootState) => ({ userUuid: getUserUuid(state) });

export const ProjectInputComponent = connect(mapStateToProps)(
    class ProjectInputComponent extends React.Component<GenericInputProps & DispatchProp & HasUserUuid & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    }, ProjectInputComponentState> {
        state: ProjectInputComponentState = {
            open: false,
        };

        render() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));

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
            this.props.input.onChange(this.state.project);
        }

        setProject = (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => {
            if ('kind' in data && data.kind === ResourceKind.PROJECT) {
                this.setState({ project: data });
            } else {
                this.setState({ project: undefined });
            }
        }

        invalid = () => (!this.state.project || this.state.project.writableBy.indexOf(this.props.userUuid) === -1);

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
            return this.state.open ? <Dialog
                open={this.state.open}
                onClose={this.closeDialog}
                fullWidth
                data-cy="choose-a-project-dialog"
                maxWidth='md'>
                <DialogTitle>Choose a project</DialogTitle>
                <DialogContent>
                    <ProjectsTreePicker
                        pickerId={this.props.commandInput.id}
                        options={this.props.options}
                        toggleItemActive={this.setProject} />
                </DialogContent>
                <DialogActions>
                    <Button onClick={this.closeDialog}>Cancel</Button>
                    <Button
                        disabled={this.invalid()}
                        variant='contained'
                        color='primary'
                        onClick={this.submit}>Ok</Button>
                </DialogActions>
            </Dialog> : null;
        }

    });
