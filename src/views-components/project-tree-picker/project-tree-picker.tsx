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
import { treePickerActions } from "~/store/tree-picker/tree-picker-actions";
import { ListItemTextIcon } from "~/components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon, FavoriteIcon, ProjectsIcon, ShareMeIcon } from "~/components/icon/icon";
import { createTreePickerNode } from "~/store/tree-picker/tree-picker";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { FilterBuilder } from "~/services/api/filter-builder";
import { WrappedFieldProps } from 'redux-form';

type ProjectTreePickerProps = Pick<TreePickerProps, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen'>;

const mapDispatchToProps = (dispatch: Dispatch, props: { onChange: (projectUuid: string) => void }): ProjectTreePickerProps => ({
    onContextMenu: () => { return; },
    toggleItemActive: (nodeId, status, pickerId) => {
        getNotSelectedTreePickerKind(pickerId)
            .forEach(pickerId => dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId: '', pickerId })));
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId, pickerId }));

        props.onChange(nodeId);
    },
    toggleItemOpen: (nodeId, status, pickerId) => {
        dispatch<any>(toggleItemOpen(nodeId, status, pickerId));
    }
});

const toggleItemOpen = (nodeId: string, status: TreeItemStatus, pickerId: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (status === TreeItemStatus.INITIAL) {
            if (pickerId === TreePickerId.PROJECTS) {
                dispatch<any>(loadProjectTreePickerProjects(nodeId));
            } else if (pickerId === TreePickerId.FAVORITES) {
                dispatch<any>(loadFavoriteTreePickerProjects(nodeId === services.authService.getUuid() ? '' : nodeId));
            } else {
                // TODO: load sharedWithMe
            }
        } else {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId }));
        }
    };

const getNotSelectedTreePickerKind = (pickerId: string) => {
    return [TreePickerId.PROJECTS, TreePickerId.FAVORITES, TreePickerId.SHARED_WITH_ME].filter(nodeId => nodeId !== pickerId);
};

export enum TreePickerId {
    PROJECTS = 'Projects',
    SHARED_WITH_ME = 'Shared with me',
    FAVORITES = 'Favorites'
}

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


// TODO: move action creator to store directory
export const loadProjectTreePickerProjects = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerId.PROJECTS }));

        const ownerUuid = nodeId.length === 0 ? services.authService.getUuid() || '' : nodeId;

        const filters = new FilterBuilder()
            .addEqual('ownerUuid', ownerUuid)
            .getFilters();

        const { items } = await services.projectService.list({ filters });

        dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerId.PROJECTS));
    };

export const loadFavoriteTreePickerProjects = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const parentId = services.authService.getUuid() || '';

        if (nodeId === '') {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId: parentId, pickerId: TreePickerId.FAVORITES }));
            const { items } = await services.favoriteService.list(parentId);

            dispatch<any>(receiveTreePickerData(parentId, items as ProjectResource[], TreePickerId.FAVORITES));
        } else {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerId.FAVORITES }));
            const filters = new FilterBuilder()
                .addEqual('ownerUuid', nodeId)
                .getFilters();

            const { items } = await services.projectService.list({ filters });

            dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerId.FAVORITES));
        }

    };

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
        name={item.data.name}
        isActive={item.active}
        hasMargin={true} />;


// TODO: move action creator to store directory
export const receiveTreePickerData = (nodeId: string, projects: ProjectResource[], pickerId: string) =>
    (dispatch: Dispatch) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            nodeId,
            nodes: projects.map(project => createTreePickerNode({ nodeId: project.uuid, value: project })),
            pickerId,
        }));

        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId }));
    };

export const ProjectTreePickerField = (props: WrappedFieldProps) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <ProjectTreePicker onChange={handleChange(props)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) => (value: string) =>
    props.input.value === value
        ? props.input.onChange('')
        : props.input.onChange(value);

