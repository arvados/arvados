// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { TreeNode, initTreeNode, getNodeDescendants, TreeNodeStatus, getNode, TreePickerId, Tree } from '~/models/tree';
import { createCollectionFilesTree } from "~/models/collection-file";
import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { getUserUuid } from "~/common/getuser";
import { ServiceRepository } from '~/services/services';
import { FilterBuilder } from '~/services/api/filter-builder';
import { pipe, values } from 'lodash/fp';
import { ResourceKind } from '~/models/resource';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { getTreePicker, TreePicker } from './tree-picker';
import { ProjectsTreePickerItem } from '~/views-components/projects-tree-picker/generic-projects-tree-picker';
import { OrderBuilder } from '~/services/api/order-builder';
import { ProjectResource } from '~/models/project';
import { mapTree } from '../../models/tree';
import { LinkResource, LinkClass } from "~/models/link";
import { mapTreeValues } from "~/models/tree";
import { sortFilesTree } from "~/services/collection-service/collection-service-files-response";
import { GroupClass, GroupResource } from "~/models/group";

export const treePickerActions = unionize({
    LOAD_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string }>(),
    LOAD_TREE_PICKER_NODE_SUCCESS: ofType<{ id: string, nodes: Array<TreeNode<any>>, pickerId: string }>(),
    APPEND_TREE_PICKER_NODE_SUBTREE: ofType<{ id: string, subtree: Tree<any>, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_COLLAPSE: ofType<{ id: string, pickerId: string }>(),
    ACTIVATE_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string, relatedTreePickers?: string[] }>(),
    DEACTIVATE_TREE_PICKER_NODE: ofType<{ pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_SELECTION: ofType<{ id: string, pickerId: string }>(),
    SELECT_TREE_PICKER_NODE: ofType<{ id: string | string[], pickerId: string }>(),
    DESELECT_TREE_PICKER_NODE: ofType<{ id: string | string[], pickerId: string }>(),
    EXPAND_TREE_PICKER_NODES: ofType<{ ids: string[], pickerId: string }>(),
    RESET_TREE_PICKER: ofType<{ pickerId: string }>()
});

export type TreePickerAction = UnionOf<typeof treePickerActions>;

export const getProjectsTreePickerIds = (pickerId: string) => ({
    home: `${pickerId}_home`,
    shared: `${pickerId}_shared`,
    favorites: `${pickerId}_favorites`,
    publicFavorites: `${pickerId}_publicFavorites`
});

export const getAllNodes = <Value>(pickerId: string, filter = (node: TreeNode<Value>) => true) => (state: TreePicker) =>
    pipe(
        () => values(getProjectsTreePickerIds(pickerId)),

        ids => ids
            .map(id => getTreePicker<Value>(id)(state)),

        trees => trees
            .map(getNodeDescendants(''))
            .reduce((allNodes, nodes) => allNodes.concat(nodes), []),

        allNodes => allNodes
            .reduce((map, node) =>
                filter(node)
                    ? map.set(node.id, node)
                    : map, new Map<string, TreeNode<Value>>())
            .values(),

        uniqueNodes => Array.from(uniqueNodes),
    )();
export const getSelectedNodes = <Value>(pickerId: string) => (state: TreePicker) =>
    getAllNodes<Value>(pickerId, node => node.selected)(state);

export const initProjectsTreePicker = (pickerId: string) =>
    async (dispatch: Dispatch, _: () => RootState, services: ServiceRepository) => {
        const { home, shared, favorites, publicFavorites } = getProjectsTreePickerIds(pickerId);
        dispatch<any>(initUserProject(home));
        dispatch<any>(initSharedProject(shared));
        dispatch<any>(initFavoritesProject(favorites));
        dispatch<any>(initPublicFavoritesProject(publicFavorites));
    };

interface ReceiveTreePickerDataParams<T> {
    data: T[];
    extractNodeData: (value: T) => { id: string, value: T, status?: TreeNodeStatus };
    id: string;
    pickerId: string;
}

export const receiveTreePickerData = <T>(params: ReceiveTreePickerDataParams<T>) =>
    (dispatch: Dispatch) => {
        const { data, extractNodeData, id, pickerId, } = params;
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id,
            nodes: data.map(item => initTreeNode(extractNodeData(item))),
            pickerId,
        }));
        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
    };

interface LoadProjectParams {
    id: string;
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
    includeFilterGroups?: boolean;
    loadShared?: boolean;
}
export const loadProject = (params: LoadProjectParams) =>
    async (dispatch: Dispatch, _: () => RootState, services: ServiceRepository) => {
        const { id, pickerId, includeCollections = false, includeFiles = false, includeFilterGroups = false, loadShared = false } = params;

        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId }));

        const filters = pipe(
            (fb: FilterBuilder) => includeCollections
                ? fb.addIsA('uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION])
                : fb.addIsA('uuid', [ResourceKind.PROJECT]),
            fb => fb.getFilters(),
        )(new FilterBuilder());

        const { items } = await services.groupsService.contents(loadShared ? '' : id, { filters, excludeHomeProject: loadShared || undefined });

        dispatch<any>(receiveTreePickerData<GroupContentsResource>({
            id,
            pickerId,
            data: items.filter((item) => {
                    if (!includeFilterGroups && (item as GroupResource).groupClass && (item as GroupResource).groupClass === GroupClass.FILTER) {
                        return false;
                    }
                    return true;
                }),
            extractNodeData: item => ({
                id: item.uuid,
                value: item,
                status: item.kind === ResourceKind.PROJECT
                    ? TreeNodeStatus.INITIAL
                    : includeFiles
                        ? TreeNodeStatus.INITIAL
                        : TreeNodeStatus.LOADED
            }),
        }));
    };

export const loadCollection = (id: string, pickerId: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId }));

        const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(getState().treePicker);
        if (picker) {

            const node = getNode(id)(picker);
            if (node && 'kind' in node.value && node.value.kind === ResourceKind.COLLECTION) {
                const files = await services.collectionService.files(node.value.portableDataHash);
                const tree = createCollectionFilesTree(files);
                const sorted = sortFilesTree(tree);
                const filesTree = mapTreeValues(services.collectionService.extendFileURL)(sorted);

                dispatch(
                    treePickerActions.APPEND_TREE_PICKER_NODE_SUBTREE({
                        id,
                        pickerId,
                        subtree: mapTree(node => ({ ...node, status: TreeNodeStatus.LOADED }))(filesTree)
                    }));

                dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
            }
        }
    };


export const initUserProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const uuid = getUserUuid(getState());
        if (uuid) {
            dispatch(receiveTreePickerData({
                id: '',
                pickerId,
                data: [{ uuid, name: 'Projects' }],
                extractNodeData: value => ({
                    id: value.uuid,
                    status: TreeNodeStatus.INITIAL,
                    value,
                }),
            }));
        }
    };
export const loadUserProject = (pickerId: string, includeCollections = false, includeFiles = false) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const uuid = getUserUuid(getState());
        if (uuid) {
            dispatch(loadProject({ id: uuid, pickerId, includeCollections, includeFiles }));
        }
    };

export const SHARED_PROJECT_ID = 'Shared with me';
export const initSharedProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(receiveTreePickerData({
            id: '',
            pickerId,
            data: [{ uuid: SHARED_PROJECT_ID, name: SHARED_PROJECT_ID }],
            extractNodeData: value => ({
                id: value.uuid,
                status: TreeNodeStatus.INITIAL,
                value,
            }),
        }));
    };

export const FAVORITES_PROJECT_ID = 'Favorites';
export const initFavoritesProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(receiveTreePickerData({
            id: '',
            pickerId,
            data: [{ uuid: FAVORITES_PROJECT_ID, name: FAVORITES_PROJECT_ID }],
            extractNodeData: value => ({
                id: value.uuid,
                status: TreeNodeStatus.INITIAL,
                value,
            }),
        }));
    };

export const PUBLIC_FAVORITES_PROJECT_ID = 'Public Favorites';
export const initPublicFavoritesProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(receiveTreePickerData({
            id: '',
            pickerId,
            data: [{ uuid: PUBLIC_FAVORITES_PROJECT_ID, name: PUBLIC_FAVORITES_PROJECT_ID }],
            extractNodeData: value => ({
                id: value.uuid,
                status: TreeNodeStatus.INITIAL,
                value,
            }),
        }));
    };

interface LoadFavoritesProjectParams {
    pickerId: string;
    includeCollections?: boolean;
    includeFiles?: boolean;
}

export const loadFavoritesProject = (params: LoadFavoritesProjectParams,
    options: { showOnlyOwned: boolean, showOnlyWritable: boolean } = { showOnlyOwned: true, showOnlyWritable: false }) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const { pickerId, includeCollections = false, includeFiles = false } = params;
        const uuid = getUserUuid(getState());
        if (uuid) {
            const filters = pipe(
                (fb: FilterBuilder) => includeCollections
                    ? fb.addIsA('head_uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION])
                    : fb.addIsA('head_uuid', [ResourceKind.PROJECT]),
                fb => fb.getFilters(),
            )(new FilterBuilder());

            const { items } = await services.favoriteService.list(uuid, { filters }, options.showOnlyOwned);

            dispatch<any>(receiveTreePickerData<GroupContentsResource>({
                id: 'Favorites',
                pickerId,
                data: items.filter((item) => {
                    if (options.showOnlyWritable && (item as GroupResource).writableBy && (item as GroupResource).writableBy.indexOf(uuid) === -1) {
                        return false;
                    }

                    return true;
                }),
                extractNodeData: item => ({
                    id: item.uuid,
                    value: item,
                    status: item.kind === ResourceKind.PROJECT
                        ? TreeNodeStatus.INITIAL
                        : includeFiles
                            ? TreeNodeStatus.INITIAL
                            : TreeNodeStatus.LOADED
                }),
            }));
        }
    };

export const loadPublicFavoritesProject = (params: LoadFavoritesProjectParams) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const { pickerId, includeCollections = false, includeFiles = false } = params;
        const uuidPrefix = getState().auth.config.uuidPrefix;
        const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;

        const filters = pipe(
            (fb: FilterBuilder) => includeCollections
                ? fb.addIsA('head_uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION])
                : fb.addIsA('head_uuid', [ResourceKind.PROJECT]),
            fb => fb
                .addEqual('link_class', LinkClass.STAR)
                .addEqual('owner_uuid', publicProjectUuid)
                .getFilters(),
        )(new FilterBuilder());

        const { items } = await services.linkService.list({ filters });

        dispatch<any>(receiveTreePickerData<LinkResource>({
            id: 'Public Favorites',
            pickerId,
            data: items,
            extractNodeData: item => ({
                id: item.headUuid,
                value: item,
                status: item.headKind === ResourceKind.PROJECT
                    ? TreeNodeStatus.INITIAL
                    : includeFiles
                        ? TreeNodeStatus.INITIAL
                        : TreeNodeStatus.LOADED
            }),
        }));
    };

export const receiveTreePickerProjectsData = (id: string, projects: ProjectResource[], pickerId: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id,
            nodes: projects.map(project => initTreeNode({ id: project.uuid, value: project })),
            pickerId,
        }));

        dispatch(treePickerActions.TOGGLE_TREE_PICKER_NODE_COLLAPSE({ id, pickerId }));
    };

export const loadProjectTreePickerProjects = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId: TreePickerId.PROJECTS }));


        const ownerUuid = id.length === 0 ? getUserUuid(getState()) || '' : id;
        const { items } = await services.projectService.list(buildParams(ownerUuid));

        dispatch<any>(receiveTreePickerProjectsData(id, items, TreePickerId.PROJECTS));
    };

export const loadFavoriteTreePickerProjects = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const parentId = getUserUuid(getState()) || '';

        if (id === '') {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: parentId, pickerId: TreePickerId.FAVORITES }));
            const { items } = await services.favoriteService.list(parentId);
            dispatch<any>(receiveTreePickerProjectsData(parentId, items as ProjectResource[], TreePickerId.FAVORITES));
        } else {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId: TreePickerId.FAVORITES }));
            const { items } = await services.projectService.list(buildParams(id));
            dispatch<any>(receiveTreePickerProjectsData(id, items, TreePickerId.FAVORITES));
        }

    };

export const loadPublicFavoriteTreePickerProjects = (id: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const parentId = getUserUuid(getState()) || '';

        if (id === '') {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id: parentId, pickerId: TreePickerId.PUBLIC_FAVORITES }));
            const { items } = await services.favoriteService.list(parentId);
            dispatch<any>(receiveTreePickerProjectsData(parentId, items as ProjectResource[], TreePickerId.PUBLIC_FAVORITES));
        } else {
            dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId: TreePickerId.PUBLIC_FAVORITES }));
            const { items } = await services.projectService.list(buildParams(id));
            dispatch<any>(receiveTreePickerProjectsData(id, items, TreePickerId.PUBLIC_FAVORITES));
        }

    };

const buildParams = (ownerUuid: string) => {
    return {
        filters: new FilterBuilder()
            .addEqual('owner_uuid', ownerUuid)
            .getFilters(),
        order: new OrderBuilder<ProjectResource>()
            .addAsc('name')
            .getOrder()
    };
};
