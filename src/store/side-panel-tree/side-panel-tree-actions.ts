// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { treePickerActions } from "~/store/tree-picker/tree-picker-actions";
import { createTreePickerNode, TreePickerNode } from '~/store/tree-picker/tree-picker';
import { RootState } from '../store';
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '~/common/api/filter-builder';
import { resourcesActions } from '../resources/resources-actions';
import { getTreePicker, TreePicker } from '../tree-picker/tree-picker';
import { TreeItemStatus } from "~/components/tree/tree";
import { getNodeAncestors, getNodeValue } from '~/models/tree';
import { ProjectResource } from '~/models/project';

export enum SidePanelTreeCategory {
    PROJECTS = 'Projects',
    SHARED_WITH_ME = 'Shared with me',
    WORKFLOWS = 'Workflows',
    RECENT_OPEN = 'Recent open',
    FAVORITES = 'Favorites',
    TRASH = 'Trash'
}

export const SIDE_PANEL_TREE = 'sidePanelTree';

export const getSidePanelTree = (treePicker: TreePicker) =>
    getTreePicker<ProjectResource | string>(SIDE_PANEL_TREE)(treePicker);

export const getSidePanelTreeBranch = (uuid: string) => (treePicker: TreePicker): Array<TreePickerNode<ProjectResource | string>> => {
    const tree = getSidePanelTree(treePicker);
    if (tree) {
        const ancestors = getNodeAncestors(uuid)(tree).map(node => node.value);
        const node = getNodeValue(uuid)(tree);
        if (node) {
            return [...ancestors, node];
        }
    }
    return [];
};

const SIDE_PANEL_CATEGORIES = [
    SidePanelTreeCategory.SHARED_WITH_ME,
    SidePanelTreeCategory.WORKFLOWS,
    SidePanelTreeCategory.RECENT_OPEN,
    SidePanelTreeCategory.FAVORITES,
    SidePanelTreeCategory.TRASH,
];

export const isSidePanelTreeCategory = (id: string) => SIDE_PANEL_CATEGORIES.some(category => category === id);

export const initSidePanelTree = () =>
    (dispatch: Dispatch, getState: () => RootState, { authService }: ServiceRepository) => {
        const rootProjectUuid = authService.getUuid() || '';
        const nodes = SIDE_PANEL_CATEGORIES.map(nodeId => createTreePickerNode({ nodeId, value: nodeId }));
        const projectsNode = createTreePickerNode({ nodeId: rootProjectUuid, value: SidePanelTreeCategory.PROJECTS });
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            nodeId: '',
            pickerId: SIDE_PANEL_TREE,
            nodes: [projectsNode, ...nodes]
        }));
        SIDE_PANEL_CATEGORIES.forEach(category => {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
                nodeId: category,
                pickerId: SIDE_PANEL_TREE,
                nodes: []
            }));
        });
    };

export const loadSidePanelTreeProjects = (projectUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ nodeId: projectUuid, pickerId: SIDE_PANEL_TREE }));
        const params = {
            filters: new FilterBuilder()
                .addEqual('ownerUuid', projectUuid)
                .getFilters()
        };
        const { items } = await services.projectService.list(params);
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            nodeId: projectUuid,
            pickerId: SIDE_PANEL_TREE,
            nodes: items.map(item => createTreePickerNode({ nodeId: item.uuid, value: item })),
        }));
        dispatch(resourcesActions.SET_RESOURCES(items));
    };

export const activateSidePanelTreeItem = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const node = getSidePanelTreeNode(nodeId)(getState());
        if (node && !node.selected) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId, pickerId: SIDE_PANEL_TREE }));
        }
        if (!isSidePanelTreeCategory(nodeId)) {
            await dispatch<any>(activateSidePanelTreeProject(nodeId));
        }
    };

export const activateSidePanelTreeProject = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const node = getSidePanelTreeNode(nodeId)(getState());
        if (node && node.status !== TreeItemStatus.LOADED) {
            await dispatch<any>(loadSidePanelTreeProjects(nodeId));
            if (node.collapsed) {
                dispatch<any>(toggleSidePanelTreeItemCollapse(nodeId));
            }
        } else if (node === undefined) {
            await dispatch<any>(activateSidePanelTreeBranch(nodeId));
        }
    };

export const activateSidePanelTreeBranch = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await services.ancestorsService.ancestors(nodeId, services.authService.getUuid() || '');
        for (const ancestor of ancestors) {
            await dispatch<any>(loadSidePanelTreeProjects(ancestor.uuid));
        }
        for (const ancestor of ancestors) {
            dispatch<any>(toggleSidePanelTreeItemCollapse(ancestor.uuid));
        }
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_SELECT({ nodeId, pickerId: SIDE_PANEL_TREE }));
    };

const getSidePanelTreeNode = (nodeId: string) => (state: RootState) => {
    const sidePanelTree = getTreePicker(SIDE_PANEL_TREE)(state.treePicker);
    return sidePanelTree
        ? getNodeValue(nodeId)(sidePanelTree)
        : undefined;
};

export const toggleSidePanelTreeItemCollapse = (nodeId: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ nodeId, pickerId: SIDE_PANEL_TREE }));
    };
