// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { Typography } from "@material-ui/core";
import { TreePicker, TreePickerProps } from "../tree-picker/tree-picker";
import { TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { ProjectResource } from "~/models/project";
import { treePickerActions, loadProjectTreePickerProjects, loadFavoriteTreePickerProjects } from "~/store/tree-picker/tree-picker-actions";
import { ListItemTextIcon } from "~/components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon, FavoriteIcon, ProjectsIcon, ShareMeIcon } from "~/components/icon/icon";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { WrappedFieldProps } from 'redux-form';
import { TreePickerId } from '~/models/tree';
import { ProjectsTreePicker } from '~/views-components/projects-tree-picker/projects-tree-picker';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { PickerIdProp } from '~/store/tree-picker/picker-id';

type ProjectTreePickerProps = Pick<TreePickerProps<ProjectResource>, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen' | 'toggleItemSelection'>;

const mapDispatchToProps = (dispatch: Dispatch, props: { onChange: (projectUuid: string) => void }): ProjectTreePickerProps => ({
    onContextMenu: () => { return; },
    toggleItemActive: (_, { id }, pickerId) => {
        getNotSelectedTreePickerKind(pickerId)
            .forEach(pickerId => dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: '', pickerId })));
        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id, pickerId }));

        props.onChange(id);
    },
    toggleItemOpen: (_, { id, status }, pickerId) => {
        dispatch<any>(toggleItemOpen(id, status, pickerId));
    },
    toggleItemSelection: (_, { id }, pickerId) => {
        dispatch<any>(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECTION({ id, pickerId }));
    },
});

const toggleItemOpen = (id: string, status: TreeItemStatus, pickerId: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (status === TreeItemStatus.INITIAL) {
            if (pickerId === TreePickerId.PROJECTS) {
                dispatch<any>(loadProjectTreePickerProjects(id));
            } else if (pickerId === TreePickerId.FAVORITES) {
                dispatch<any>(loadFavoriteTreePickerProjects(id === services.authService.getUuid() ? '' : id));
            } else {
                // TODO: load sharedWithMe
            }
        } else {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
        }
    };

const getNotSelectedTreePickerKind = (pickerId: string) => {
    return [TreePickerId.PROJECTS, TreePickerId.FAVORITES, TreePickerId.SHARED_WITH_ME].filter(nodeId => nodeId !== pickerId);
};

export const ProjectTreePicker = connect(undefined, mapDispatchToProps)((props: ProjectTreePickerProps) =>
    <div style={{ display: 'flex', flexDirection: 'column' }}>
        <Typography variant='caption' style={{ flexShrink: 0 }}>
            Select a project
        </Typography>
        <div style={{ flexGrow: 1, overflow: 'auto' }}>
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.PROJECTS} />
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.SHARED_WITH_ME} />
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.FAVORITES} />
        </div>
    </div>);

const getProjectPickerIcon = (item: TreeItem<ProjectResource>) => {
    switch (item.data.name) {
        case TreePickerId.FAVORITES:
            return FavoriteIcon;
        case TreePickerId.PROJECTS:
            return ProjectsIcon;
        case TreePickerId.SHARED_WITH_ME:
            return ShareMeIcon;
        default:
            return ProjectIcon;
    }
};

const renderTreeItem = (item: TreeItem<ProjectResource>) =>
    <ListItemTextIcon
        icon={getProjectPickerIcon(item)}
        name={typeof item.data === 'string' ? item.data : item.data.name}
        isActive={item.active}
        hasMargin={true} />;

export const ProjectTreePickerField = (props: WrappedFieldProps & PickerIdProp) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectsTreePicker
            pickerId={props.pickerId}
            toggleItemActive={handleChange(props)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) =>
    (_: any, { id }: TreeItem<ProjectsTreePickerItem>) =>
        props.input.onChange(id);
