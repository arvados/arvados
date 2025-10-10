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
import { loadProject } from 'store/workbench/workbench-actions';
import { runProcessPanelActions } from 'store/run-process-panel/run-process-panel-actions';
import { isUserResource } from 'models/user';

export type RunWfProjectCommandInputParameter = GenericCommandInputParameter<ProjectResource, ProjectResource>;

const isUndefined: any = (value?: ProjectResource) => (value === undefined);

interface ProjectInputProps {
    required: boolean;
    input: RunWfProjectCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

type DialogContentCssRules = 'root' | 'pickerWrapper';

export const RunWfProjectInput = ({ required, input, options }: ProjectInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={ProjectInputComponent as any}
        format={format}
        validate={required ? isUndefined : undefined}
        {...{
            options,
            required
        }} />;

const format = (value?: ProjectResource) => value ? value.name : '';

interface ProjectInputComponentState {
    open: boolean;
    hasBeenOpened: boolean;
    defaultProject?: ProjectResource;
    originalProject?: ProjectResource;
    selectedProject?: ProjectResource;
    targetProject?: ProjectResource;
}

type ProjectInputComponentProps = {
    userUuid: string | undefined;
    userRootProject: ProjectResource | undefined;
    defaultTargetProject: ProjectResource | undefined;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
    required?: boolean;
}

interface HasUserUuid {
    userUuid: string;
}

const mapStateToProps = (state: RootState): Pick<ProjectInputComponentProps, 'userUuid' | 'userRootProject' | 'defaultTargetProject' > => {
    const userUuid = getUserUuid(state)
    const userRootProject = getResource<ProjectResource>(userUuid)(state.resources);
    const defaultTargetProject = getResource<ProjectResource>(state.runProcessPanel.processOwnerUuid)(state.resources)
    return {
        userUuid,
        userRootProject,
        defaultTargetProject,
    }
};

const ProjectInputComponent = connect(mapStateToProps)(
    class ProjectInputComponent extends React.Component<GenericInputProps & DispatchProp & HasUserUuid & ProjectInputComponentProps, ProjectInputComponentState> {

        state: ProjectInputComponentState = {
            open: false,
            hasBeenOpened: false,
            defaultProject: undefined, // defined in redux as the current project where the workflow will run
            originalProject: undefined, // selected project when the dialog was opened
            selectedProject: undefined, // current project selected in the dialog
            targetProject: undefined, // set on submit when dialog closes
        };

        componentDidMount() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));
            const project = this.getDefaultProject();
            // set initial selected project
            if (!this.state.selectedProject && project) {
                this.setState({
                    defaultProject: project,
                    selectedProject: project,
                });
            }
            // load user root project if not already loaded
            if (this.props.userUuid && (!this.props.userRootProject || !isUserResource(this.props.userRootProject)  || !('firstName' in this.props.userRootProject))) {
                this.props.dispatch<any>(loadProject(this.props.userUuid));
            }
            // open dialog automatically when input mounts
            if (this.state.hasBeenOpened === false) {
                this.setState({ open: true, hasBeenOpened: true });
            }
        }

        componentDidUpdate(prevProps: ProjectInputComponentProps, prevState: ProjectInputComponentState) {
            // set target project if not already set
            if (!this.state.targetProject) {
                const project = this.getDefaultProject();
                if (project) {
                    this.setState({ targetProject: project });
                }
            }
            // set default project if user root project changes (e.g. when user root project loads)
            if (this.props.userRootProject && prevProps.userRootProject !== this.props.userRootProject) {
                const project = this.getDefaultProject();
                this.setState({
                    defaultProject: project,
                    selectedProject: project,
                    targetProject: project,
                });
            }
            // ensures that the target & selected project are set if page reloads
            if (prevProps.defaultTargetProject !== this.props.defaultTargetProject) {
                const project = this.getDefaultProject();
                this.setState({ selectedProject: project, targetProject: project });
            }
        }

        componentWillUnmount(): void {
            this.props.dispatch<any>(runProcessPanelActions.SET_PROCESS_OWNER_UUID(''));
            this.setState({ targetProject: undefined });
        }

        getDefaultProject = () => {
            const { userUuid, userRootProject, defaultTargetProject } = this.props;
            if (defaultTargetProject?.canWrite) {
                return defaultTargetProject;
            }
            const isTargetUser = (defaultTargetProject as any)?.kind === ResourceKind.USER;
            const isTargetUserThisUser = isTargetUser && defaultTargetProject?.uuid === userUuid;
            if (isTargetUserThisUser) {
                if (defaultTargetProject) {
                    return defaultTargetProject;
                }
                return userRootProject;
            }
            return defaultTargetProject || userRootProject
        }

        render() {
            return <>
                {this.renderInput()}
                <DialogComponent
                    targetProject={this.state.targetProject}
                    open={this.state.open}
                    closeDialog={this.closeDialog}
                    setProject={this.setProject}
                    submit={this.submit}
                    invalid={this.invalid}
                    commandInput={this.props.commandInput}
                    options={this.props.options}
                />
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
            if (this.state.selectedProject) {
                if (this.state.selectedProject.kind === ResourceKind.PROJECT || this.state.selectedProject.kind === ResourceKind.USER) {
                    this.props.dispatch<any>(runProcessPanelActions.SET_PROCESS_OWNER_UUID(this.state.selectedProject.uuid));
                    this.setState({ targetProject: this.state.selectedProject });
                }
                if (this.state.originalProject && this.state.selectedProject.uuid !== this.state.originalProject.uuid) {
                    this.props.input.onChange(this.state.selectedProject);
                }
            }
        }

        setProject = (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => {
            if ('kind' in data){
                if (data.kind === ResourceKind.PROJECT) {
                    this.setState({ selectedProject: data });
                } else if (data.kind === ResourceKind.USER) {
                    this.setState({ selectedProject: this.props.userRootProject });
                }
            } else {
                this.setState({ selectedProject: undefined });
            }
        }

        getDisplayName(item: ProjectsTreePickerItem | undefined): string {
            if (item === undefined) {
                return '';
            }
            if ('kind' in item && item.kind === ResourceKind.USER) {
                return `${item.firstName} ${item.lastName} (root project)`;
            }
            if ('name' in item) {
                return item.name;
            } else {
                return '';
            }
        }


        invalid = () => (!this.state.selectedProject || !this.state.selectedProject.canWrite);

        renderInput() {
            return <GenericInput
                component={props =>
                    <Input
                        readOnly
                        fullWidth
                        data-cy='run-wf-project-input'
                        value={this.getDisplayName(this.state.targetProject)}
                        error={props.meta.touched && !!props.meta.error}
                        disabled={props.commandInput.disabled}
                        onClick={!this.props.commandInput.disabled ? this.openDialog : undefined}
                        onKeyPress={!this.props.commandInput.disabled ? () => this.openDialog : undefined}
                        onMouseDown={(e) => e.preventDefault()} />}
                {...this.props} />;
        }
    });

const dialogContentStyles: CustomStyleRulesCallback<DialogContentCssRules> = ({ spacing }) => ({
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

const DialogComponent = withStyles(dialogContentStyles)(
    ( props: WithStyles<DialogContentCssRules> & {
        targetProject: ProjectResource | undefined,
        open: boolean,
        closeDialog: () => void,
        setProject: (_: {}, { data }: TreeItem<ProjectsTreePickerItem>) => void,
        submit: () => void,
        invalid: () => boolean,
        commandInput: GenericCommandInputParameter<ProjectResource, ProjectResource>,
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean },
    }) =>
        props.open ?
            <Dialog
                open={props.open}
                onClose={props.closeDialog}
                fullWidth
                data-cy="choose-a-project-dialog"
                maxWidth='md'>
                    <DialogTitle>Choose the project where the workflow will run</DialogTitle>
                    <DialogContent className={props.classes.root}>
                        <div className={props.classes.pickerWrapper}>
                            {props.targetProject && <ProjectsTreePicker
                                pickerId={props.commandInput.id}
                                cascadeSelection={false}
                                options={props.options}
                                project={props.targetProject}
                                currentUuids={[props.targetProject.uuid]}
                                toggleItemActive={props.setProject} />}
                        </div>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={props.closeDialog} data-cy='run-wf-project-picker-cancel-button'>
                            Cancel
                        </Button>
                        <Button
                            data-cy='run-wf-project-picker-ok-button'
                            disabled={props.invalid()}
                            variant='contained'
                            color='primary'
                            onClick={props.submit}>Ok</Button>
                    </DialogActions>
            </Dialog> : null
        );
