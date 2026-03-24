// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Typography } from "@mui/material";
import { TreeItem } from "components/tree/tree";
import { WrappedFieldProps } from 'redux-form';
import { ProjectsTreePicker } from 'views-components/projects-tree-picker/projects-tree-picker';
import { ProjectsTreePickerItem } from 'store/tree-picker/tree-picker-middleware';
import { PickerIdProp } from 'store/tree-picker/picker-id';
import { FileOperationLocation, getFileOperationLocation, SEARCH_PROJECT_ID_PREFIX } from "store/tree-picker/tree-picker-actions";
import { connect } from "react-redux";
import { Dispatch } from "redux";

type TreePickerDialogProps = {
    pickerId: string;
    currentUuids?: string[];
    setSelectedProject: (uuid: string) => void;
}

export const ProjectTreePickerField = (props: WrappedFieldProps & PickerIdProp) =>
    <div style={{ display: 'flex', minHeight: 0, flexDirection: 'column' }}>
        <div style={{ flexBasis: '960px', flexShrink: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
            <ProjectsTreePicker
                pickerId={props.pickerId}
                toggleItemActive={handleChange(props)}
                cascadeSelection={false}
                options={{ showOnlyOwned: false, showOnlyWritable: true }} />
            {props.meta.dirty && props.meta.error &&
                <Typography variant='caption' color='error'>
                    {props.meta.error}
                </Typography>}
        </div>
    </div>;

export const ProjectTreePickerDialogField = (props: TreePickerDialogProps) =>
    <div style={{ display: 'flex', minHeight: 0, flexDirection: 'column' }}>
        <div style={{ flexBasis: '960px', flexShrink: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
            <ProjectsTreePicker
                pickerId={props.pickerId}
                currentUuids={props.currentUuids}
                toggleItemActive={(_: any, { id }) => props.setSelectedProject(id)}
                cascadeSelection={false}
                options={{ showOnlyOwned: false, showOnlyWritable: true }} />
        </div>
    </div>;

const handleChange = (props: WrappedFieldProps) =>
    (_: any, { id }: TreeItem<ProjectsTreePickerItem>) => {
        if (id.startsWith(SEARCH_PROJECT_ID_PREFIX)) {
            props.input.onChange(id.slice(SEARCH_PROJECT_ID_PREFIX.length));
        } else {
            props.input.onChange(id);
        }
    }

type ProjectsTreePickerActionProps = {
    getFileOperationLocation: (item: ProjectsTreePickerItem) => Promise<FileOperationLocation | undefined>;
}

const projectsTreePickerMapDispatchToProps = (dispatch: Dispatch): ProjectsTreePickerActionProps => ({
    getFileOperationLocation: (item: ProjectsTreePickerItem) => dispatch<any>(getFileOperationLocation(item)),
});

type ProjectsTreePickerCombinedProps = ProjectsTreePickerActionProps & WrappedFieldProps & PickerIdProp;

export const DirectoryTreePickerField = connect(null, projectsTreePickerMapDispatchToProps)(
    class DirectoryTreePickerFieldComponent extends React.Component<ProjectsTreePickerCombinedProps> {

        handleDirectoryChange = (props: WrappedFieldProps) =>
            async (_: any, { data }: TreeItem<ProjectsTreePickerItem>) => {
                const location = await this.props.getFileOperationLocation(data);
                props.input.onChange(location || '');
            }

        render() {
            return <div style={{ display: 'flex', minHeight: 0, flexDirection: 'column' }}>
                <div style={{ flexBasis: '275px', flexShrink: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
                    <ProjectsTreePicker
                        currentUuids={[this.props.input.value.uuid]}
                        pickerId={this.props.pickerId}
                        toggleItemActive={this.handleDirectoryChange(this.props)}
                        cascadeSelection={false}
                        options={{ showOnlyOwned: false, showOnlyWritable: true }}
                        includeCollections
                        includeDirectories />
                    {this.props.meta.dirty && this.props.meta.error &&
                        <Typography variant='caption' color='error'>
                            {this.props.meta.error}
                        </Typography>}
                </div>
            </div>;
        }
    });

type DirectoryTreePickerDialogFieldProps = PickerIdProp & {
    currentUuids?: string[];
    getFileOperationLocation: (item: ProjectsTreePickerItem) => Promise<FileOperationLocation | undefined>;
    handleDirectoryChange: (destination: FileOperationLocation) => void;
};

export const DirectoryTreePickerDialogField = connect(null, projectsTreePickerMapDispatchToProps)(
    (props: DirectoryTreePickerDialogFieldProps)=> {

    const handleDirectoryChange = (props: DirectoryTreePickerDialogFieldProps) =>
            async (_: any, { data }: TreeItem<ProjectsTreePickerItem>) => {
                const location: FileOperationLocation | undefined = await props.getFileOperationLocation(data);
                if (location) {
                    props.handleDirectoryChange(location);
                } else {
                    props.handleDirectoryChange({} as FileOperationLocation);
                }
            }

    return (
        <div style={{ display: 'flex', minHeight: 0, flexDirection: 'column' }}>
            <div style={{ flexBasis: '960px', flexShrink: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
                <ProjectsTreePicker
                    currentUuids={props.currentUuids}
                    pickerId={props.pickerId}
                    toggleItemActive={handleDirectoryChange(props)}
                    cascadeSelection={false}
                    options={{ showOnlyOwned: false, showOnlyWritable: true }}
                    includeCollections
                    includeDirectories />
            </div>
        </div>
        );
    }
);
