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

export type SearchProjectCommandInputParameter = GenericCommandInputParameter<ProjectResource, ProjectResource>;

const isUndefined: any = (value?: ProjectResource) => (value === undefined);

interface ProjectInputProps {
    required: boolean;
    input: SearchProjectCommandInputParameter;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

type DialogContentCssRules = 'root' | 'pickerWrapper';

export const SearchProjectInput = ({ required, input, options }: ProjectInputProps) =>
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
    project?: ProjectResource;
}

interface HasUserUuid {
    userUuid: string;
}

const mapStateToProps = (state: RootState) => ({ userUuid: getUserUuid(state) });

const ProjectInputComponent = connect(mapStateToProps)(
    class ProjectInputComponent extends React.Component<GenericInputProps & DispatchProp & HasUserUuid & {
        options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
        required?: boolean;
    }, ProjectInputComponentState> {
        state: ProjectInputComponentState = {
            open: false,
        };

        componentDidMount() {
            this.props.dispatch<any>(
                initProjectsTreePicker(this.props.commandInput.id));
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

        invalid = () => (!this.state.project || !this.state.project.canWrite);

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
                            disabled={this.invalid()}
                            variant='contained'
                            color='primary'
                            onClick={this.submit}>Ok</Button>
                    </DialogActions>
                </Dialog> : null
        );

    });
