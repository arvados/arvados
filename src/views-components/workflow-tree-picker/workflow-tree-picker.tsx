// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { Typography } from "@material-ui/core";
import { MainWorkflowTreePicker, MainWorkflowTreePickerProps } from "./main-workflow-tree-picker";
import { TreeItem, TreeItemStatus } from "~/components/tree/tree";
import { ProjectResource } from "~/models/project";
import { workflowTreePickerActions } from "~/store/workflow-tree-picker/workflow-tree-picker-actions";
import { ListItemTextIcon } from "~/components/list-item-text-icon/list-item-text-icon";
import { ProjectIcon, FavoriteIcon, ProjectsIcon, ShareMeIcon, CollectionIcon } from '~/components/icon/icon';
import { createTreePickerNode } from "~/store/tree-picker/tree-picker";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { FilterBuilder } from "~/services/api/filter-builder";
import { WrappedFieldProps } from 'redux-form';
import { ResourceKind } from '~/models/resource';
import { GroupContentsResource } from '~/services/groups-service/groups-service';

type WorkflowTreePickerProps = Pick<MainWorkflowTreePickerProps, 'onContextMenu' | 'toggleItemActive' | 'toggleItemOpen'>;

const mapDispatchToProps = (dispatch: Dispatch, props: { onChange: (projectUuid: string) => void }): WorkflowTreePickerProps => ({
    onContextMenu: () => { return; },
    toggleItemActive: (nodeId, status, pickerId) => {
        getNotSelectedTreePickerKind(pickerId)
            .forEach(pickerId => dispatch(workflowTreePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId: '', pickerId })));
        dispatch(workflowTreePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId, pickerId }));

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
                dispatch<any>(loadProjectTreePicker(nodeId));
            } else if (pickerId === TreePickerId.FAVORITES) {
                dispatch<any>(loadFavoriteTreePicker(nodeId === services.authService.getUuid() ? '' : nodeId));
            } else {
                // TODO: load sharedWithMe
            }
        } else {
            dispatch(workflowTreePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId }));
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

export const WorkflowTreePicker = connect(undefined, mapDispatchToProps)((props: WorkflowTreePickerProps) =>
    <div style={{ display: 'flex', flexDirection: 'column' }}>
        <Typography variant='caption' style={{ flexShrink: 0 }}>
            Select a project
        </Typography>
        <div style={{ flexGrow: 1, overflow: 'auto' }}>
            <MainWorkflowTreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.PROJECTS} />
            <MainWorkflowTreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.SHARED_WITH_ME} />
            <MainWorkflowTreePicker {...props} render={renderTreeItem} pickerId={TreePickerId.FAVORITES} />
        </div>
    </div>);

export const loadProjectTreePicker = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(workflowTreePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerId.PROJECTS }));

        const ownerUuid = nodeId.length === 0 ? services.authService.getUuid() || '' : nodeId;

        const filters = new FilterBuilder()
            .addIsA("uuid", [ResourceKind.PROJECT, ResourceKind.COLLECTION])
            .addEqual('ownerUuid', ownerUuid)
            .getFilters();

        const { items } = await services.groupsService.contents(ownerUuid, { filters });

        dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerId.PROJECTS));
    };

export const loadFavoriteTreePicker = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const parentId = services.authService.getUuid() || '';

        if (nodeId === '') {
            dispatch(workflowTreePickerActions.LOAD_TREE_PICKER_NODE({ nodeId: parentId, pickerId: TreePickerId.FAVORITES }));
            const { items } = await services.favoriteService.list(parentId);

            dispatch<any>(receiveTreePickerData(parentId, items as ProjectResource[], TreePickerId.FAVORITES));
        } else {
            dispatch(workflowTreePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerId.FAVORITES }));
            const filters = new FilterBuilder()
                .addEqual('ownerUuid', nodeId)
                .getFilters();

            const { items } = await services.groupsService.contents(parentId, { filters });

            dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerId.FAVORITES));
        }
    };

const getProjectPickerIcon = (item: TreeItem<ProjectResource>) => {
    console.log(item);
    switch (item.data.name) {
        case TreePickerId.FAVORITES:
            return FavoriteIcon;
        case TreePickerId.PROJECTS:
            return ProjectsIcon;
        case TreePickerId.SHARED_WITH_ME:
            return ShareMeIcon;
        default:
            return getResourceIcon(item);
    }
};

const getResourceIcon = (item: TreeItem<GroupContentsResource>) => {
    switch (item.data.kind) {
        case ResourceKind.COLLECTION:
            return CollectionIcon;
        case ResourceKind.PROJECT:
            return ProjectsIcon;
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


export const receiveTreePickerData = (nodeId: string, items: GroupContentsResource[], pickerId: string) =>
    (dispatch: Dispatch) => {
        dispatch(workflowTreePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            nodeId,
            nodes: items.map(item => createTreePickerNode({ nodeId: item.uuid, value: item })),
            pickerId,
        }));

        dispatch(workflowTreePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId }));
    };

export const WorkflowTreePickerField = (props: WrappedFieldProps) =>
    <div style={{ height: '200px', display: 'flex', flexDirection: 'column' }}>
        <WorkflowTreePicker onChange={handleChange(props)} />
        {props.meta.dirty && props.meta.error &&
            <Typography variant='caption' color='error'>
                {props.meta.error}
            </Typography>}
    </div>;

const handleChange = (props: WrappedFieldProps) => (value: string) =>
    props.input.value === value
        ? props.input.onChange('')
        : props.input.onChange(value);

