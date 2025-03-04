// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { Field } from 'redux-form';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Input, Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import {
    GenericCommandInputParameter
} from 'models/workflow';
import { GenericInput, GenericInputProps } from './generic-input';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { initProjectsTreePicker } from 'store/tree-picker/tree-picker-actions';
import { TreeItem } from 'components/tree/tree';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { RootState } from 'store/store';
import { getUserUuid } from 'common/getuser';
import { getResource } from 'store/resources/resources';

export type ProjectCommandInputParameter = GenericCommandInputParameter<ProjectResource, ProjectResource>;

const isUndefined: any = (value?: ProjectResource) => (value === undefined);

export interface ProjectInputProps {
    required: boolean;
    input: ProjectCommandInputParameter;
    isRunProcessForm?: boolean;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

type DialogContentCssRules = 'root' | 'pickerWrapper';

export const ProjectInput = ({ required, input, options, isRunProcessForm }: ProjectInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={ProjectInputComponent as any}
        format={format}
        validate={required ? isUndefined : undefined}
        isRunProcessForm={isRunProcessForm}
        {...{
            options,
            required,
        }} />;

const format = (value?: ProjectResource) => value ? value.name : '';

type ProjectInputComponentProps = {
    isRunProcessForm?: boolean;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    required?: boolean;
    defaultOwner: ProjectResource;
}

interface ProjectInputComponentState {
    open: boolean;
    project?: ProjectResource;
    hasProjectBeenSet: boolean;
    defaultOwner?: ProjectResource;
}

interface HasUserUuid {
    userUuid: string;
}

const mapStateToProps = (state: RootState) => ({
    userUuid: getUserUuid(state),
    defaultOwner: getResource(state.runProcessPanel.processOwnerUuid)(state.resources) });

export const ProjectInputComponent = connect(mapStateToProps)(
    class ProjectInputComponent extends React.Component<GenericInputProps & DispatchProp & HasUserUuid & ProjectInputComponentProps, ProjectInputComponentState> {
        state: ProjectInputComponentState = {
            open: false,
            hasProjectBeenSet: false,
        };

        componentDidMount() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));
        }

        componentDidUpdate(prevProps: any, prevState: ProjectInputComponentState) {
            if (!!prevState.project === false && !!this.state.project && this.state.hasProjectBeenSet === false) {
                this.setState({ hasProjectBeenSet: true });
            }
        }

        render() {
            return <>
                {this.renderInput()}
                <this.dialog />
            </>;
        }

        openDialog = () => {
            this.componentDidMount();
            this.setState({ open: true });
        }

        closeDialog = () => {
            this.setState({ open: false, hasProjectBeenSet: true });
        }

        submit = () => {
            this.closeDialog();
            this.props.input.onChange(this.state.project);
        }

        setProject = (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => {
            if ('kind' in data && (data.kind === ResourceKind.PROJECT || data.kind === ResourceKind.USER)) {
                this.setState({ project: data as ProjectResource });
            } else {
                this.setState({ project: undefined });
            }
        }

        getDisplayName(item: ProjectsTreePickerItem): string {
            if ('kind' in item && item.kind === ResourceKind.USER) {
                return `${item.firstName} ${item.lastName} (root project)`;
            }
            if ('name' in item) {
                return item.name;
            } else {
                return '';
            }
        }

        renderInput() {
            const { open, project, hasProjectBeenSet } = this.state;
            const { isRunProcessForm, defaultOwner } = this.props;
            if (isRunProcessForm && open === false && !project && !hasProjectBeenSet) this.openDialog();
            return <GenericInput
                component={props =>
                    <Input
                        readOnly
                        fullWidth
                        value={(defaultOwner && this.getDisplayName(defaultOwner)) || props.input.value}
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
                this.state.open ? <Dialog
                                      open={this.state.open}
                                      onClose={this.closeDialog}
                                      fullWidth
                                      data-cy="choose-a-project-dialog"
                                      maxWidth='md'>
                    <DialogTitle>Choose a project</DialogTitle>
                    <DialogContent className={classes.root}>
                        <div className={classes.pickerWrapper}>
                            <ProjectsTreePicker
                                pickerId={this.props.commandInput.id}
                                cascadeSelection={false}
                                options={this.props.options}
                                toggleItemActive={this.setProject} />
                        </div>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={this.closeDialog}>Cancel</Button>
                        <Button
                            variant='contained'
                            color='primary'
                            onClick={this.submit}>Ok</Button>
                    </DialogActions>
                </Dialog> : null
        );

    });
