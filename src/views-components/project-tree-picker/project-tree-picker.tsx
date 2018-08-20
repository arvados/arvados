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
import { FilterBuilder } from "~/common/api/filter-builder";
import { mockProjectResource } from "~/models/test-utils";

type ProjectTreePickerProps = Pick<TreePickerProps, 'toggleItemActive' | 'toggleItemOpen'>;

const mapDispatchToProps = (dispatch: Dispatch, props: { onChange: (projectUuid: string) => void }): ProjectTreePickerProps => ({
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
            if (pickerId === TreePickerKind.PROJECTS) {
                dispatch<any>(loadProjectTreePickerProjects(nodeId));
            } else if (pickerId === TreePickerKind.FAVORITES) {
                dispatch<any>(loadFavoriteTreePickerProjects(nodeId === services.authService.getUuid() ? '' : nodeId));
            } else {
                // TODO: load sharedWithMe
            }
        } else {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId }));
        }
    };

const getNotSelectedTreePickerKind = (pickerId: string) => {
    return [TreePickerKind.PROJECTS, TreePickerKind.FAVORITES, TreePickerKind.SHARED_WITH_ME].filter(nodeId => nodeId !== pickerId);
};

export enum TreePickerKind {
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
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerKind.PROJECTS} />
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerKind.SHARED_WITH_ME} />
            <TreePicker {...props} render={renderTreeItem} pickerId={TreePickerKind.FAVORITES} />
        </div>
    </div>);


// TODO: move action creator to store directory
export const loadProjectTreePickerProjects = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerKind.PROJECTS }));

        const ownerUuid = nodeId.length === 0 ? services.authService.getUuid() || '' : nodeId;

        const filters = new FilterBuilder()
            .addEqual('ownerUuid', ownerUuid)
            .getFilters();

        const { items } = await services.projectService.list({ filters });

        dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerKind.PROJECTS));
    };

export const loadFavoriteTreePickerProjects = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const parentId = services.authService.getUuid() || '';

        if (nodeId === '') {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId: parentId, pickerId: TreePickerKind.FAVORITES }));
            const { items } = await services.favoriteService.list(parentId);

            dispatch<any>(receiveTreePickerData(parentId, items as ProjectResource[], TreePickerKind.FAVORITES));
        } else {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId, pickerId: TreePickerKind.FAVORITES }));
            const filters = new FilterBuilder()
                .addEqual('ownerUuid', nodeId)
                .getFilters();

            const { items } = await services.projectService.list({ filters });

            dispatch<any>(receiveTreePickerData(nodeId, items, TreePickerKind.FAVORITES));
        }

    };

const getProjectPickerIcon = (item: TreeItem<ProjectResource>) => {
    switch (item.data.name) {
        case TreePickerKind.FAVORITES:
            return FavoriteIcon;
        case TreePickerKind.PROJECTS:
            return ProjectsIcon;
        case TreePickerKind.SHARED_WITH_ME:
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

export const initPickerProjectTree = () => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const uuid = services.authService.getUuid();

    dispatch<any>(getPickerTreeProjects(uuid));
    dispatch<any>(getSharedWithMeProjectsPickerTree(uuid));
    dispatch<any>(getFavoritesProjectsPickerTree(uuid));
};

const getPickerTreeProjects = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.PROJECTS);
};

const getSharedWithMeProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.SHARED_WITH_ME);
};

const getFavoritesProjectsPickerTree = (uuid: string = '') => {
    return getProjectsPickerTree(uuid, TreePickerKind.FAVORITES);
};

const getProjectsPickerTree = (uuid: string, kind: string) => {
    return receiveTreePickerData(
        '',
        [mockProjectResource({ uuid, name: kind })],
        kind
    );
};

