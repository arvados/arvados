// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getTreePicker, TreePicker } from "store/tree-picker/tree-picker";
import { getNode, getNodeAncestorsIds, initTreeNode, TreeNodeStatus } from "models/tree";
import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { treePickerActions } from "store/tree-picker/tree-picker-actions";
import { FilterBuilder } from "services/api/filter-builder";
import { OrderBuilder } from "services/api/order-builder";
import { ProjectResource } from "models/project";
import { resourcesActions } from "store/resources/resources-actions";
import { SEARCH_BAR_ADVANCED_FORM_PICKER_ID } from "store/search-bar/search-bar-actions";

const getSearchBarTreeNode = (id: string) => (treePicker: TreePicker) => {
    const searchTree = getTreePicker(SEARCH_BAR_ADVANCED_FORM_PICKER_ID)(treePicker);
    return searchTree
        ? getNode(id)(searchTree)
        : undefined;
};

export const loadSearchBarTreeProjects = (projectUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const treePicker = getTreePicker(SEARCH_BAR_ADVANCED_FORM_PICKER_ID)(getState().treePicker);
        const node = treePicker ? getNode(projectUuid)(treePicker) : undefined;
        if (node || projectUuid === '') {
            await dispatch<any>(loadSearchBarProject(projectUuid));
        }
    };

export const getSearchBarTreeNodeAncestorsIds = (id: string) => (treePicker: TreePicker) => {
    const searchTree = getTreePicker(SEARCH_BAR_ADVANCED_FORM_PICKER_ID)(treePicker);
    return searchTree
        ? getNodeAncestorsIds(id)(searchTree)
        : [];
};

export const activateSearchBarTreeBranch = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const ancestors = await services.ancestorsService.ancestors(id, userUuid);

        for (const ancestor of ancestors) {
            await dispatch<any>(loadSearchBarTreeProjects(ancestor.uuid));
        }
        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODES({
            ids: [
                ...[],
                ...ancestors.map(ancestor => ancestor.uuid)
            ],
            pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID
        }));
        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id, pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID }));
    };

export const expandSearchBarTreeItem = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const node = getSearchBarTreeNode(id)(getState().treePicker);
        if (node && !node.expanded) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID }));
        }
    };

export const activateSearchBarProject = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {


        /*const { treePicker } = getState();
        const node = getSearchBarTreeNode(id)(treePicker);
        if (node && node.status !== TreeNodeStatus.LOADED) {
            await dispatch<any>(loadSearchBarTreeProjects(id));
        } else if (node === undefined) {
            await dispatch<any>(activateSearchBarTreeBranch(id));
        }
        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODES({
            ids: getSearchBarTreeNodeAncestorsIds(id)(treePicker),
            pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID
        }));
        dispatch<any>(expandSearchBarTreeItem(id));*/
    };


const loadSearchBarProject = (projectUuid: string) =>
    async (dispatch: Dispatch, _: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: projectUuid, pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID }));
        const params = {
            filters: new FilterBuilder()
                .addEqual('owner_uuid', projectUuid)
                .getFilters(),
            order: new OrderBuilder<ProjectResource>()
                .addAsc('name')
                .getOrder()
        };
        const { items } = await services.projectService.list(params);
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: projectUuid,
            pickerId: SEARCH_BAR_ADVANCED_FORM_PICKER_ID,
            nodes: items.map(item => initTreeNode({ id: item.uuid, value: item })),
        }));
        dispatch(resourcesActions.SET_RESOURCES(items));
    };
