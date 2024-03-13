// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { treePickerActions } from "store/tree-picker/tree-picker-actions";
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from 'services/services';
import { FilterBuilder } from 'services/api/filter-builder';
import { resourcesActions } from 'store/resources/resources-actions';
import { getTreePicker, TreePicker } from 'store/tree-picker/tree-picker';
import { getNodeAncestors, getNodeAncestorsIds, getNode, TreeNode, initTreeNode, TreeNodeStatus } from 'models/tree';
import { ProjectResource } from 'models/project';
import { OrderBuilder } from 'services/api/order-builder';
import { ResourceKind } from 'models/resource';
import { CategoriesListReducer } from 'common/plugintypes';
import { pluginConfig } from 'plugins';
import { LinkClass, LinkResource } from 'models/link';
import { verifyAndUpdateLinks } from 'common/link-update-name';

export enum SidePanelTreeCategory {
    PROJECTS = 'Home Projects',
    FAVORITES = 'My Favorites',
    PUBLIC_FAVORITES = 'Public Favorites',
    SHARED_WITH_ME = 'Shared with me',
    ALL_PROCESSES = 'All Processes',
    INSTANCE_TYPES = 'Instance Types',
    SHELL_ACCESS = 'Shell Access',
    GROUPS = 'Groups',
    TRASH = 'Trash',
}

export const SIDE_PANEL_TREE = 'sidePanelTree';
const SIDEPANEL_TREE_NODE_LIMIT = 50

export const getSidePanelTree = (treePicker: TreePicker) =>
    getTreePicker<ProjectResource | string>(SIDE_PANEL_TREE)(treePicker);

export const getSidePanelTreeBranch = (uuid: string) => (treePicker: TreePicker): Array<TreeNode<ProjectResource | string>> => {
    const tree = getSidePanelTree(treePicker);
    if (tree) {
        const ancestors = getNodeAncestors(uuid)(tree);
        const node = getNode(uuid)(tree);
        if (node) {
            return [...ancestors, node];
        }
    }
    return [];
};

let SIDE_PANEL_CATEGORIES: string[] = [
    SidePanelTreeCategory.PROJECTS,
    SidePanelTreeCategory.FAVORITES,
    SidePanelTreeCategory.PUBLIC_FAVORITES,
    SidePanelTreeCategory.SHARED_WITH_ME,
    SidePanelTreeCategory.ALL_PROCESSES,
    SidePanelTreeCategory.INSTANCE_TYPES,
    SidePanelTreeCategory.SHELL_ACCESS,
    SidePanelTreeCategory.GROUPS,
    SidePanelTreeCategory.TRASH
];

const reduceCatsFn: (a: string[],
    b: CategoriesListReducer) => string[] = (a, b) => b(a);

SIDE_PANEL_CATEGORIES = pluginConfig.sidePanelCategories.reduce(reduceCatsFn, SIDE_PANEL_CATEGORIES);

export const isSidePanelTreeCategory = (id: string) => SIDE_PANEL_CATEGORIES.some(category => category === id);


export const initSidePanelTree = () =>
    (dispatch: Dispatch, getState: () => RootState, { authService }: ServiceRepository) => {
        const rootProjectUuid = getUserUuid(getState());
        if (!rootProjectUuid) { return; }
        const nodes = SIDE_PANEL_CATEGORIES.map(id => {
            if (id === SidePanelTreeCategory.PROJECTS) {
                return initTreeNode({ id: rootProjectUuid, value: SidePanelTreeCategory.PROJECTS });
            } else {
                return initTreeNode({ id, value: id });
            }
        });
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: '',
            pickerId: SIDE_PANEL_TREE,
            nodes
        }));
        SIDE_PANEL_CATEGORIES.forEach(category => {
                if (category !== SidePanelTreeCategory.PROJECTS && category !== SidePanelTreeCategory.FAVORITES && category !== SidePanelTreeCategory.PUBLIC_FAVORITES ) {
                dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
                    id: category,
                    pickerId: SIDE_PANEL_TREE,
                    nodes: []
                }));
            }
        });
    };

export const loadSidePanelTreeProjects = (projectUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const treePicker = getTreePicker(SIDE_PANEL_TREE)(getState().treePicker);
        const node = treePicker ? getNode(projectUuid)(treePicker) : undefined;
        if (projectUuid === SidePanelTreeCategory.PUBLIC_FAVORITES) {
            const unverifiedPubFaves = await dispatch<any>(loadPublicFavoritesTree());
            verifyAndUpdateLinkNames(unverifiedPubFaves, dispatch, getState, services);
        } else if (projectUuid === SidePanelTreeCategory.FAVORITES) {
            const unverifiedFaves = await dispatch<any>(loadFavoritesTree());
            await setFaves(unverifiedFaves, dispatch, getState, services);
            verifyAndUpdateLinkNames(unverifiedFaves, dispatch, getState, services);
        } else if (node || projectUuid !== '') {
            await dispatch<any>(loadProject(projectUuid));
        }
    };

const loadProject = (projectUuid: string) =>
    async (dispatch: Dispatch, _: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: projectUuid, pickerId: SIDE_PANEL_TREE }));
        const params = {
            filters: new FilterBuilder()
                .addEqual('owner_uuid', projectUuid)
                .getFilters(),
            order: new OrderBuilder<ProjectResource>()
                .addDesc('createdAt')
                .getOrder(),
            limit: SIDEPANEL_TREE_NODE_LIMIT,
        };

        const { items } = await services.projectService.list(params);

        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: projectUuid,
            pickerId: SIDE_PANEL_TREE,
            nodes: items.map(item => initTreeNode({ id: item.uuid, value: item })),
        }));
        dispatch(resourcesActions.SET_RESOURCES(items));
    };

export const loadFavoritesTree = () => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: SidePanelTreeCategory.FAVORITES, pickerId: SIDE_PANEL_TREE }));

    const params = {
        filters: new FilterBuilder()
            .addEqual('link_class', LinkClass.STAR)
            .addEqual('tail_uuid', getUserUuid(getState()))
            .addEqual('tail_kind', ResourceKind.USER)
            .getFilters(),
        order: new OrderBuilder<ProjectResource>().addDesc('createdAt').getOrder(),
        limit: SIDEPANEL_TREE_NODE_LIMIT,
    };

    const { items } = await services.linkService.list(params);

    dispatch(
        treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: SidePanelTreeCategory.FAVORITES,
            pickerId: SIDE_PANEL_TREE,
            nodes: items.map(item => initTreeNode({ id: item.headUuid, value: item })),
        })
    );

    return items;
};

const setFaves = async(links: LinkResource[], dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {

    const uuids = links.map(it => it.headUuid);
    const groupItems: any = await services.groupsService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .getFilters()
    });
    const collectionItems: any = await services.collectionService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .getFilters()
    });
    const processItems: any = await services.containerRequestService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .getFilters()
    });
    const responseItems = groupItems.items.concat(collectionItems.items).concat(processItems.items);

    //setting resources here so they won't be re-fetched in validation step
    dispatch(resourcesActions.SET_RESOURCES(responseItems));
};

const verifyAndUpdateLinkNames = async (links: LinkResource[], dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const verfifiedLinks = await verifyAndUpdateLinks(links, dispatch, getState, services);

    dispatch(
        treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: SidePanelTreeCategory.FAVORITES,
            pickerId: SIDE_PANEL_TREE,
            nodes: verfifiedLinks.map(item => initTreeNode({ id: item.headUuid, value: item })),
        })
    );
};

export const loadPublicFavoritesTree = () => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: SidePanelTreeCategory.PUBLIC_FAVORITES, pickerId: SIDE_PANEL_TREE }));

    const uuidPrefix = getState().auth.config.uuidPrefix;
    const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;
    const typeFilters = [ResourceKind.COLLECTION, ResourceKind.CONTAINER_REQUEST, ResourceKind.GROUP, ResourceKind.WORKFLOW];

    const params = {
        filters: new FilterBuilder()
            .addEqual('link_class', LinkClass.STAR)
            .addEqual('owner_uuid', publicProjectUuid)
            .addIsA('head_uuid', typeFilters)
            .getFilters(),
        order: new OrderBuilder<ProjectResource>().addDesc('createdAt').getOrder(),
        limit: SIDEPANEL_TREE_NODE_LIMIT,
    };

    const { items } = await services.linkService.list(params);

    const uuids = items.map(it => it.headUuid);
    const groupItems: any = await services.groupsService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .addIsA("uuid", typeFilters)
            .getFilters()
    });
    const collectionItems: any = await services.collectionService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .addIsA("uuid", typeFilters)
            .getFilters()
    });
    const processItems: any = await services.containerRequestService.list({
        select: ['uuid', 'name'],
        filters: new FilterBuilder()
            .addIn("uuid", uuids)
            .addIsA("uuid", typeFilters)
            .getFilters()
    });

    const responseItems = groupItems.items.concat(collectionItems.items).concat(processItems.items);

    const filteredItems = items.filter(item => responseItems.some(responseItem => responseItem.uuid === item.headUuid));

    dispatch(
        treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id: SidePanelTreeCategory.PUBLIC_FAVORITES,
            pickerId: SIDE_PANEL_TREE,
            nodes: filteredItems.map(item => initTreeNode({ id: item.headUuid, value: item })),
        })
    );

    //setting resources here so they won't be re-fetched in validation step
    dispatch(resourcesActions.SET_RESOURCES(responseItems));

    return filteredItems;
};

export const activateSidePanelTreeItem = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const node = getSidePanelTreeNode(id)(getState().treePicker);
        if (node && !node.active) {
        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id, pickerId: SIDE_PANEL_TREE }));
    }
    if (!isSidePanelTreeCategory(id)) {
            await dispatch<any>(activateSidePanelTreeProject(id));
        }
    };

export const activateSidePanelTreeProject = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const { treePicker } = getState();
        const node = getSidePanelTreeNode(id)(treePicker);
        if (node && node.status !== TreeNodeStatus.LOADED) {
            await dispatch<any>(loadSidePanelTreeProjects(id));
        } else if (node === undefined) {
            await dispatch<any>(activateSidePanelTreeBranch(id));
        }
        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODES({
            ids: getSidePanelTreeNodeAncestorsIds(id)(treePicker),
            pickerId: SIDE_PANEL_TREE
        }));
        dispatch<any>(expandSidePanelTreeItem(id));
    };

export const activateSidePanelTreeBranch = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (!userUuid) { return; }
        const ancestors = await services.ancestorsService.ancestors(id, userUuid);
        for (const ancestor of ancestors) {
            await dispatch<any>(loadSidePanelTreeProjects(ancestor.uuid));
        }
        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODES({
            ids: ancestors.map(ancestor => ancestor.uuid),
            pickerId: SIDE_PANEL_TREE
        }));
        dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id, pickerId: SIDE_PANEL_TREE }));
    };

export const toggleSidePanelTreeItemCollapse = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const node = getSidePanelTreeNode(id)(getState().treePicker);
        if (node && node.status === TreeNodeStatus.INITIAL) {
            await dispatch<any>(loadSidePanelTreeProjects(node.id));
        }
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId: SIDE_PANEL_TREE }));
    };

export const expandSidePanelTreeItem = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const node = getSidePanelTreeNode(id)(getState().treePicker);
        if (node && !node.expanded) {
            dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId: SIDE_PANEL_TREE }));
        }
    };

export const getSidePanelTreeNode = (id: string) => (treePicker: TreePicker) => {
    const sidePanelTree = getTreePicker(SIDE_PANEL_TREE)(treePicker);
    return sidePanelTree
        ? getNode(id)(sidePanelTree)
        : undefined;
};

export const getSidePanelTreeNodeAncestorsIds = (id: string) => (treePicker: TreePicker) => {
    const sidePanelTree = getTreePicker(SIDE_PANEL_TREE)(treePicker);
    return sidePanelTree
        ? getNodeAncestorsIds(id)(sidePanelTree)
        : [];
};
