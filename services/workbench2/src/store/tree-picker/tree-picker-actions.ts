// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "common/unionize";
import { TreeNode, initTreeNode, getNodeDescendants, TreeNodeStatus, getNode, TreePickerId, Tree, setNode, createTree, getNodeDescendantsIds } from 'models/tree';
import { CollectionFileType, createCollectionFilesTree, getCollectionResourceCollectionUuid } from "models/collection-file";
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from 'services/services';
import { FilterBuilder } from 'services/api/filter-builder';
import { pipe, values } from 'lodash/fp';
import { ResourceKind, ResourceObjectType, extractUuidObjectType, COLLECTION_PDH_REGEX } from 'models/resource';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { getTreePicker, TreePicker} from './tree-picker';
import { ProjectsTreePickerItem } from './tree-picker-middleware';
import { OrderBuilder } from 'services/api/order-builder';
import { ProjectResource } from 'models/project';
import { mapTree } from '../../models/tree';
import { LinkResource, LinkClass } from "models/link";
import { mapTreeValues } from "models/tree";
import { sortFilesTree } from "services/collection-service/collection-service-files-response";
import { GroupClass, GroupResource } from "models/group";
import { CollectionResource } from "models/collection";
import { getResource } from "store/resources/resources";
import { updateResources } from "store/resources/resources-actions";
import { SnackbarKind, snackbarActions } from "store/snackbar/snackbar-actions";
import { call, put, takeEvery, takeLatest, getContext, select, all } from "redux-saga/effects";
import { TreeItemWeight, TreeItemWithWeight } from "components/tree/tree";

export const treePickerActions = unionize({
    LOAD_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string }>(),
    LOAD_TREE_PICKER_NODE_SUCCESS: ofType<{ id: string, nodes: Array<TreeNode<any>>, pickerId: string }>(),
    APPEND_TREE_PICKER_NODE_SUBTREE: ofType<{ id: string, subtree: Tree<any>, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_COLLAPSE: ofType<{ id: string, pickerId: string }>(),
    EXPAND_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string }>(),
    EXPAND_TREE_PICKER_NODE_ANCESTORS: ofType<{ id: string, pickerId: string }>(),
    ACTIVATE_TREE_PICKER_NODE: ofType<{ id: string, pickerId: string, relatedTreePickers?: string[] }>(),
    DEACTIVATE_TREE_PICKER_NODE: ofType<{ pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_SELECTION: ofType<{ id: string, pickerId: string, cascade: boolean }>(),
    SELECT_TREE_PICKER_NODE: ofType<{ id: string | string[], pickerId: string, cascade: boolean }>(),
    DESELECT_TREE_PICKER_NODE: ofType<{ id: string | string[], pickerId: string, cascade: boolean }>(),
    EXPAND_TREE_PICKER_NODES: ofType<{ ids: string[], pickerId: string }>(),
    RESET_TREE_PICKER: ofType<{ pickerId: string }>()
});

export type TreePickerAction = UnionOf<typeof treePickerActions>;

export interface LoadProjectParams {
    includeCollections?: boolean;
    includeDirectories?: boolean;
    includeFiles?: boolean;
    includeFilterGroups?: boolean;
    options?: { showOnlyOwned: boolean; showOnlyWritable: boolean; };
}

export const treePickerSearchActions = unionize({
    SET_TREE_PICKER_PROJECT_SEARCH: ofType<{ pickerId: string, projectSearchValue: string }>(),
    SET_TREE_PICKER_COLLECTION_FILTER: ofType<{ pickerId: string, collectionFilterValue: string }>(),
    SET_TREE_PICKER_LOAD_PARAMS: ofType<{ pickerId: string, params: LoadProjectParams }>(),
});

export type TreePickerSearchAction = UnionOf<typeof treePickerSearchActions>;

export const treePickerSearchSagas = unionize({
    SET_PROJECT_SEARCH: ofType<{ pickerId: string, projectSearchValue: string }>(),
    SET_COLLECTION_FILTER: ofType<{ pickerMainId: string, collectionFilterValue: string }>(),
    APPLY_COLLECTION_FILTER: ofType<{ pickerId: string }>(),
    LOAD_PROJECT: ofType<LoadProjectParamsWithId>(),
    LOAD_SEARCH: ofType<LoadProjectParamsWithId>(),
    LOAD_FAVORITES_PROJECT: ofType<LoadFavoritesProjectParams>(),
    LOAD_PUBLIC_FAVORITES_PROJECT: ofType<LoadFavoritesProjectParams>(),
    REFRESH_TREE_PICKER: ofType<{ pickerId: string }>(),
});

export function* setTreePickerProjectSearchWatcher() {
    // Race conditions are handled in loadSearchWatcher so takeEvery is used here to avoid confusion
    yield takeEvery(treePickerSearchSagas.tags.SET_PROJECT_SEARCH, setTreePickerProjectSearchSaga);
}

function* setTreePickerProjectSearchSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.SET_PROJECT_SEARCH,
    payload: typeof treePickerSearchSagas._Record.SET_PROJECT_SEARCH,
}) {
    try {
        const { pickerId , projectSearchValue } = payload;
        const state: RootState = yield select();
        const searchChanged = state.treePickerSearch.projectSearchValues[pickerId] !== projectSearchValue;

        if (searchChanged) {
            yield put(treePickerSearchActions.SET_TREE_PICKER_PROJECT_SEARCH(payload));
            const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(state.treePicker);
            if (picker) {
                const loadParams = state.treePickerSearch.loadProjectParams[pickerId];
                // Put is non-blocking so race-condition prevention is handled by the loadSearchWatcher
                yield put(treePickerSearchSagas.LOAD_SEARCH({
                    ...loadParams,
                    id: SEARCH_PROJECT_ID,
                    pickerId,
                }));
            }
        }
    } catch (e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to search`, kind: SnackbarKind.ERROR }));
    }
}

/**
 * Race-free collection filter saga as long as it's invoked through SET_COLLECTION_FILTER
 */
export function* setTreePickerCollectionFilterWatcher() {
    yield takeLatest(treePickerSearchSagas.tags.SET_COLLECTION_FILTER, setTreePickerCollectionFilterSaga);
}

function* setTreePickerCollectionFilterSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.SET_COLLECTION_FILTER,
    payload: typeof treePickerSearchSagas._Record.SET_COLLECTION_FILTER,
}) {
    try {
        const state: RootState = yield select();
        const { pickerMainId , collectionFilterValue } = payload;
        const pickerRootItemIds = Object.values(getProjectsTreePickerIds(pickerMainId));

        const changedRootItemIds = pickerRootItemIds.filter((pickerRootId) =>
            state.treePickerSearch.collectionFilterValues[pickerRootId] !== collectionFilterValue
        );

        yield all(pickerRootItemIds.map(pickerId =>
            put(treePickerSearchActions.SET_TREE_PICKER_COLLECTION_FILTER({
                pickerId,
                collectionFilterValue,
            }))
        ));

        yield all(changedRootItemIds.map(pickerId =>
            call(applyCollectionFilterSaga, {
                type: treePickerSearchSagas.tags.APPLY_COLLECTION_FILTER,
                payload: { pickerId }
            })
        ));
    } catch (e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to search`, kind: SnackbarKind.ERROR }));
    } finally {
        // Optionally handle cleanup when task cancelled
        // if (yield cancelled()) {}
    }
}

/**
 * Only meant to be called synchronously via call from other sagas that implement takeLatest
 */
function* applyCollectionFilterSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.APPLY_COLLECTION_FILTER,
    payload: typeof treePickerSearchSagas._Record.APPLY_COLLECTION_FILTER,
}) {
    try {
        const state: RootState = yield select();
        const { pickerId } = payload;
        if (state.treePickerSearch.projectSearchValues[pickerId] !== "") {
            yield call(refreshTreePickerSaga, {
                type: treePickerSearchSagas.tags.REFRESH_TREE_PICKER,
                payload: { pickerId }
            });
        } else {
            const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(state.treePicker);
            if (picker) {
                const loadParams = state.treePickerSearch.loadProjectParams[pickerId];
                yield call(loadProjectSaga, {
                    type: treePickerSearchSagas.tags.LOAD_PROJECT,
                    payload: {
                        ...loadParams,
                        id: SEARCH_PROJECT_ID,
                        pickerId,
                }});
            }
        }
    } catch (e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to search`, kind: SnackbarKind.ERROR }));
    }
}

export const getProjectsTreePickerIds = (pickerId: string) => ({
    home: `${pickerId}_home`,
    shared: `${pickerId}_shared`,
    favorites: `${pickerId}_favorites`,
    publicFavorites: `${pickerId}_publicFavorites`,
    search: `${pickerId}_search`,
});

export const SEARCH_PROJECT_ID_PREFIX = "search-";

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

interface TreePickerPreloadParams {
    selectedItemUuids: string[];
    includeDirectories: boolean;
    includeFiles: boolean;
    multi: boolean;
}

export const initProjectsTreePicker = (pickerId: string, preloadParams?: TreePickerPreloadParams) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { home, shared, favorites, publicFavorites, search } = getProjectsTreePickerIds(pickerId);
        dispatch<any>(initUserProject(home));
        dispatch<any>(initSharedProject(shared));
        dispatch<any>(initFavoritesProject(favorites));
        dispatch<any>(initPublicFavoritesProject(publicFavorites));
        dispatch<any>(initSearchProject(search));

        if (preloadParams && preloadParams.selectedItemUuids.length) {
            await dispatch<any>(loadInitialValue(
                preloadParams.selectedItemUuids,
                pickerId,
                preloadParams.includeDirectories,
                preloadParams.includeFiles,
                preloadParams.multi
            ));
        }
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
        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODE({ id, pickerId }));
    };

export const extractGroupContentsNodeData = (expandableCollections: boolean) => (item: GroupContentsResource & TreeItemWithWeight) => {
    if (item.uuid === "more-items-available") {
        return {
            id: item.uuid,
            value: item,
            status: TreeNodeStatus.LOADED
        }
    } else if (item.weight === TreeItemWeight.LIGHT) {
        return {
            id: SEARCH_PROJECT_ID_PREFIX+item.uuid,
            value: item,
            status: item.kind === ResourceKind.PROJECT
                  ? TreeNodeStatus.INITIAL
                  : item.kind === ResourceKind.COLLECTION && expandableCollections
                  ? TreeNodeStatus.INITIAL
                  : TreeNodeStatus.LOADED
        };
    } else {
        return { id: item.uuid,
                 value: item,
                 status: item.kind === ResourceKind.PROJECT
                       ? TreeNodeStatus.INITIAL
                       : item.kind === ResourceKind.COLLECTION && expandableCollections
                       ? TreeNodeStatus.INITIAL
                       : TreeNodeStatus.LOADED
        };
    }
};

interface LoadProjectParamsWithId extends LoadProjectParams {
    id: string;
    pickerId: string;
    loadShared?: boolean;
}

/**
 * Kicks off a picker search load that allows paralell runs
 * Used for expanding nodes
 */
export const loadProject = (params: LoadProjectParamsWithId) => (treePickerSearchSagas.LOAD_PROJECT(params));
export function* loadProjectWatcher() {
    yield takeEvery(treePickerSearchSagas.tags.LOAD_PROJECT, loadProjectSaga);
}

/**
 * Asynchronously kicks off a race-free picker search load - does not block when used this way
 */
export const loadSearch = (params: LoadProjectParamsWithId) => (treePickerSearchSagas.LOAD_SEARCH(params));
export function* loadSearchWatcher() {
    yield takeLatest(treePickerSearchSagas.tags.LOAD_SEARCH, loadProjectSaga);
}

/**
 * loadProjectSaga is used to load or refresh a project node in a tree picker
 * Errors are caught and a toast is shown if the project fails to load
 * Blocks when called directly with call(), can be composed into race-free groups
 */
function* loadProjectSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.LOAD_PROJECT,
    payload: typeof treePickerSearchSagas._Record.LOAD_PROJECT,
}) {
    try {
        const services: ServiceRepository = yield getContext("services");
        const state: RootState = yield select();

        const {
            id,
            pickerId,
            includeCollections = false,
            includeDirectories = false,
            includeFiles = false,
            includeFilterGroups = false,
            loadShared = false,
            options,
        } = payload;

        const searching = (id === SEARCH_PROJECT_ID);
        const collectionFilter = state.treePickerSearch.collectionFilterValues[pickerId];
        const projectFilter = state.treePickerSearch.projectSearchValues[pickerId];

        let filterB = new FilterBuilder();

        let includeOwners: string|undefined = undefined;

        if (id.startsWith(SEARCH_PROJECT_ID_PREFIX)) {
            return;
        }

        if (searching) {
            // opening top level search
            if (projectFilter) {
                includeOwners = "owner_uuid";
                filterB = filterB.addIsA('uuid', [ResourceKind.PROJECT]);

                const objtype = extractUuidObjectType(projectFilter);
                if (objtype === ResourceObjectType.GROUP || objtype === ResourceObjectType.USER) {
                        filterB = filterB.addEqual('uuid', projectFilter);
                }
                else {
                    filterB = filterB.addFullTextSearch(projectFilter, 'groups');
                }

            } else if (collectionFilter) {
                includeOwners = "owner_uuid";
                filterB = filterB.addIsA('uuid', [ResourceKind.COLLECTION]);

                const objtype = extractUuidObjectType(collectionFilter);
                if (objtype === ResourceObjectType.COLLECTION) {
                    filterB = filterB.addEqual('uuid', collectionFilter);
                } else if (COLLECTION_PDH_REGEX.exec(collectionFilter)) {
                    filterB = filterB.addEqual('portable_data_hash', collectionFilter);
                } else {
                    filterB = filterB.addFullTextSearch(collectionFilter, 'collections');
                }
            } else {
                return;
            }
        } else {
            // opening a folder below the top level
            if (includeCollections) {
                filterB = filterB.addIsA('uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION]);
            } else {
                filterB = filterB.addIsA('uuid', [ResourceKind.PROJECT]);
            }

            if (projectFilter) {
                filterB = filterB.addFullTextSearch(projectFilter, 'groups');
            }
            if (collectionFilter) {
                filterB = filterB.addFullTextSearch(collectionFilter, 'collections');
            }
        }

        filterB = filterB.addNotIn("collections.properties.type", ["intermediate", "log"]);

        const globalSearch = loadShared || id === SEARCH_PROJECT_ID;

        const filters = filterB.getFilters();

        // Must be under 1000
        const itemLimit = 200;

        yield put(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId }));

        let { items, included } = yield call(
            {context: services.groupsService, fn: services.groupsService.contents},
            globalSearch ? '' : id,
            {
                filters,
                excludeHomeProject: loadShared || undefined,
                limit: itemLimit+1,
                count: "none",
                include: includeOwners,
            }
        );

        if (!included) {
            includeOwners = undefined;
        }

        //let rootItems: GroupContentsResource[] | GroupContentsIncludedResource[] = items;
        let rootItems: any[] = items;

        const seen = {};

        if (includeOwners && included) {
            included = included.filter(item => {
                if (seen.hasOwnProperty(item.uuid)) {
                    return false;
                } else {
                    seen[item.uuid] = item;
                    return true;
                }
            });
            yield put(updateResources(included));

            rootItems = included;
        }

        items = items.filter(item => {
            if (seen.hasOwnProperty(item.uuid)) {
                return false;
            } else {
                seen[item.uuid] = item;
                if (!seen[item.ownerUuid] && includeOwners) {
                    rootItems.push(item);
                }
                return true;
            }
        });
        yield put(updateResources(items));

        if (items.length > itemLimit) {
            rootItems.push({
                uuid: "more-items-available-"+id,
                kind: ResourceKind.WORKFLOW,
                name: `*** Not all items listed, reduce item count with search or filter ***`,
                description: "",
                definition: "",
                ownerUuid: "",
                createdAt: "",
                modifiedByUserUuid: "",
                modifiedAt: "",
                etag: ""
            });
        }

        yield put(receiveTreePickerData<GroupContentsResource>({
            id,
            pickerId,
            data: rootItems.filter(item => {
                if (!includeFilterGroups && (item as GroupResource).groupClass && (item as GroupResource).groupClass === GroupClass.FILTER) {
                    return false;
                }

                if (options && options.showOnlyWritable && item.hasOwnProperty('frozenByUuid') && (item as ProjectResource).frozenByUuid) {
                    return false;
                }
                return true;
            }).map(item => {
                if (extractUuidObjectType(item.uuid) === ResourceObjectType.USER) {
                    return {...item,
                            uuid: item.uuid,
                            name: item['fullName'] + " Home Project",
                            weight: includeOwners ? TreeItemWeight.LIGHT : TreeItemWeight.NORMAL,
                            kind: ResourceKind.USER,
                    }
                }
                return {...item,
                        uuid: item.uuid,
                        weight: includeOwners ? TreeItemWeight.LIGHT : TreeItemWeight.NORMAL,};

            }),
            extractNodeData: extractGroupContentsNodeData(includeDirectories || includeFiles),
        }));

        if (includeOwners) {
            // Searching, we already have the
            // contents to put in the owner projects so load it up.
            const projects = {};
            items.forEach(item => {
                if (!projects.hasOwnProperty(item.ownerUuid)) {
                    projects[item.ownerUuid] = [];
                }
                projects[item.ownerUuid].push({...item, weight: TreeItemWeight.DARK});
            });
            for (const prj in projects) {
                yield put(receiveTreePickerData<GroupContentsResource>({
                    id: SEARCH_PROJECT_ID_PREFIX+prj,
                    pickerId,
                    data: projects[prj],
                    extractNodeData: extractGroupContentsNodeData(includeDirectories || includeFiles),
                }));
            }
        }
    } catch(e) {
        console.error("Failed to load project into tree picker:", e);;
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to load project`, kind: SnackbarKind.ERROR }));
    } finally {
        // Optionally handle cleanup when task cancelled
        // if (yield cancelled()) {}
    }
};

export const refreshTreePicker = (params: typeof treePickerSearchSagas._Record.REFRESH_TREE_PICKER) => (treePickerSearchSagas.REFRESH_TREE_PICKER(params));

export function* refreshTreePickerWatcher() {
    yield takeEvery(treePickerSearchSagas.tags.REFRESH_TREE_PICKER, refreshTreePickerSaga);
}

/**
 * Refreshes a single tree picker subtree
 */
function* refreshTreePickerSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.REFRESH_TREE_PICKER,
    payload: typeof treePickerSearchSagas._Record.REFRESH_TREE_PICKER,
}) {
    try {
        const state: RootState = yield select();
        const { pickerId } = payload;

        const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(state.treePicker);
        if (picker) {
            const loadParams = state.treePickerSearch.loadProjectParams[pickerId];
            yield all((getNodeDescendantsIds('')(picker)
                .reduce((acc, id) => {
                    const node = getNode(id)(picker);
                    if (node && node.status !== TreeNodeStatus.INITIAL) {
                        if (node.id.substring(6, 11) === 'tpzed' || node.id.substring(6, 11) === 'j7d0g') {
                            return acc.concat(call(loadProjectSaga, {
                                type: treePickerSearchSagas.tags.LOAD_PROJECT,
                                payload: {
                                    ...loadParams,
                                    id: node.id,
                                    pickerId,
                            }}));
                        }
                        if (node.id === SHARED_PROJECT_ID) {
                            return acc.concat(call(loadProjectSaga, {
                                type: treePickerSearchSagas.tags.LOAD_PROJECT,
                                payload: {
                                    ...loadParams,
                                    id: node.id,
                                    pickerId,
                                    loadShared: true
                            }}));
                        }
                        if (node.id === SEARCH_PROJECT_ID) {
                            return acc.concat(call(loadProjectSaga, {
                                type: treePickerSearchSagas.tags.LOAD_PROJECT,
                                payload: {
                                    ...loadParams,
                                    id: node.id,
                                    pickerId,
                            }}));
                        }
                        if (node.id === FAVORITES_PROJECT_ID) {
                            return acc.concat(call(loadFavoritesProjectSaga, {
                                type: treePickerSearchSagas.tags.LOAD_FAVORITES_PROJECT,
                                payload: {
                                    ...loadParams,
                                    pickerId,
                            }}));
                        }
                        if (node.id === PUBLIC_FAVORITES_PROJECT_ID) {
                            return acc.concat(call(loadPublicFavoritesProjectSaga, {
                                type: treePickerSearchSagas.tags.LOAD_PUBLIC_FAVORITES_PROJECT,
                                payload: {
                                    ...loadParams,
                                    pickerId,
                            }}));
                        }
                    }
                    return acc;
                }, [] as Object[])));
        }
    } catch (e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to search`, kind: SnackbarKind.ERROR }));
    }
}

export const loadCollection = (id: string, pickerId: string, includeDirectories?: boolean, includeFiles?: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE({ id, pickerId }));

        const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(getState().treePicker);
        if (picker) {

            const node = getNode(id)(picker);
            if (node && 'kind' in node.value && node.value.kind === ResourceKind.COLLECTION) {
                const files = (await services.collectionService.files(node.value.uuid))
                    .filter((file) => (
                        (includeFiles) ||
                        (includeDirectories && file.type === CollectionFileType.DIRECTORY)
                    ));
                const tree = createCollectionFilesTree(files);
                const sorted = sortFilesTree(tree);
                const filesTree = mapTreeValues(services.collectionService.extendFileURL)(sorted);

                // await tree modifications so that consumers can guarantee node presence
                await dispatch(
                    treePickerActions.APPEND_TREE_PICKER_NODE_SUBTREE({
                        id,
                        pickerId,
                        subtree: mapTree(node => ({ ...node, status: TreeNodeStatus.LOADED }))(filesTree)
                    }));

                // Expand collection root node
                dispatch(treePickerActions.EXPAND_TREE_PICKER_NODE({ id, pickerId }));
            }
        }
    };

export const HOME_PROJECT_ID = 'Home Projects';
export const initUserProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const uuid = getUserUuid(getState());
        if (uuid) {
            dispatch(receiveTreePickerData({
                id: '',
                pickerId,
                data: [{ uuid, name: HOME_PROJECT_ID }],
                extractNodeData: value => ({
                    id: value.uuid,
                    status: TreeNodeStatus.INITIAL,
                    value,
                }),
            }));
        }
    };
export const loadUserProject = (pickerId: string, includeCollections = false, includeDirectories = false, includeFiles = false, options?: { showOnlyOwned: boolean, showOnlyWritable: boolean }) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const uuid = getUserUuid(getState());
        if (uuid) {
            dispatch(loadProject({ id: uuid, pickerId, includeCollections, includeDirectories, includeFiles, options }));
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

type PickerItemPreloadData = {
    itemId: string;
    mainItemUuid: string;
    ancestors: (GroupResource | CollectionResource)[];
    isHomeProjectItem: boolean;
}

type PickerTreePreloadData = {
    tree: Tree<GroupResource | CollectionResource>;
    pickerTreeId: string;
    pickerTreeRootUuid: string;
};

export const loadInitialValue = (pickerItemIds: string[], pickerId: string, includeDirectories: boolean, includeFiles: boolean, multi: boolean,) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const homeUuid = getUserUuid(getState());

        // Request ancestor trees in paralell and save home project status
        const pickerItemsData: PickerItemPreloadData[] = await Promise.allSettled(pickerItemIds.map(async itemId => {
            const mainItemUuid = itemId.includes('/') ? itemId.split('/')[0] : itemId;

            const ancestors = (await services.ancestorsService.ancestors(mainItemUuid, ''))
            .filter(item =>
                item.kind === ResourceKind.GROUP ||
                item.kind === ResourceKind.COLLECTION
            ) as (GroupResource | CollectionResource)[];

            const isOnlyHomeProject = pickerItemIds.length === 1 && pickerItemIds[0] === homeUuid;

            if (ancestors.length === 0 && !isOnlyHomeProject) {
                return Promise.reject({item: itemId});
            }

            const isHomeProjectItem = !!((homeUuid && ancestors.some(item => item.ownerUuid === homeUuid)) || isOnlyHomeProject);

            return {
                itemId,
                mainItemUuid,
                ancestors,
                isHomeProjectItem,
            };
        })).then((res) => {
            // Show toast if any selections failed to restore
            const rejectedPromises = res.filter((promiseResult): promiseResult is PromiseRejectedResult => (promiseResult.status === 'rejected'));
            if (rejectedPromises.length) {
                rejectedPromises.forEach(item => {
                    console.error("The following item failed to load into the tree picker", item.reason);
                });
                dispatch<any>(snackbarActions.OPEN_SNACKBAR({ message: `Some selections failed to load and were removed. See console for details.`, kind: SnackbarKind.ERROR }));
            }
            // Filter out any failed promises and map to resulting preload data with ancestors
            return res.filter((promiseResult): promiseResult is PromiseFulfilledResult<PickerItemPreloadData> => (
                promiseResult.status === 'fulfilled'
            )).map(res => res.value)
        });

        // Group items to preload / ancestor data by home/shared picker and create initial Trees to preload
        const initialTreePreloadData: PickerTreePreloadData[] = [
            pickerItemsData.filter((item) => item.isHomeProjectItem),
            pickerItemsData.filter((item) => !item.isHomeProjectItem),
        ]
            .filter((items) => items.length > 0)
            .map((itemGroup) =>
                itemGroup.reduce(
                    (preloadTree, itemData) => ({
                        tree: createInitialPickerTree(
                            itemData.ancestors,
                            itemData.mainItemUuid,
                            preloadTree.tree
                        ),
                        pickerTreeId: getPickerItemTreeId(itemData, homeUuid, pickerId),
                        pickerTreeRootUuid: getPickerItemRootUuid(itemData, homeUuid),
                    }),
                    {
                        tree: createTree<GroupResource | CollectionResource>(),
                        pickerTreeId: '',
                        pickerTreeRootUuid: '',
                    } as PickerTreePreloadData
                )
            );

        // Load initial trees into corresponding picker store
        await Promise.all(initialTreePreloadData.map(preloadTree => (
            dispatch(
                treePickerActions.APPEND_TREE_PICKER_NODE_SUBTREE({
                    id: preloadTree.pickerTreeRootUuid,
                    pickerId: preloadTree.pickerTreeId,
                    subtree: preloadTree.tree,
                })
            )
        )));

        // Await loading collection before attempting to select items
        await Promise.all(pickerItemsData.map(async itemData => {
            const pickerTreeId = getPickerItemTreeId(itemData, homeUuid, pickerId);

            // Selected item resides in collection subpath
            if (itemData.itemId.includes('/')) {
                // Load collection into tree
                // loadCollection includes more than dispatched actions and must be awaited
                await dispatch(loadCollection(itemData.mainItemUuid, pickerTreeId, includeDirectories, includeFiles));
            }
            // Expand nodes down to destination
            dispatch(treePickerActions.EXPAND_TREE_PICKER_NODE_ANCESTORS({ id: itemData.itemId, pickerId: pickerTreeId }));
        }));

        // Select or activate nodes
        pickerItemsData.forEach(itemData => {
            const pickerTreeId = getPickerItemTreeId(itemData, homeUuid, pickerId);

            if (multi) {
                dispatch(treePickerActions.SELECT_TREE_PICKER_NODE({ id: itemData.itemId, pickerId: pickerTreeId, cascade: false}));
            } else {
                dispatch(treePickerActions.ACTIVATE_TREE_PICKER_NODE({ id: itemData.itemId, pickerId: pickerTreeId }));
            }
        });

        // Refresh triggers loading in all adjacent items that were not included in the ancestor tree
        await initialTreePreloadData.map(preloadTree => dispatch(treePickerSearchSagas.REFRESH_TREE_PICKER({ pickerId: preloadTree.pickerTreeId })));
    }

const getPickerItemTreeId = (itemData: PickerItemPreloadData, homeUuid: string | undefined, pickerId: string) => {
    const { home, shared } = getProjectsTreePickerIds(pickerId);
    return ((itemData.isHomeProjectItem && homeUuid) ? home : shared);
};

const getPickerItemRootUuid = (itemData: PickerItemPreloadData, homeUuid: string | undefined) => {
    return (itemData.isHomeProjectItem && homeUuid) ? homeUuid : SHARED_PROJECT_ID;
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

export const SEARCH_PROJECT_ID = 'Search all Projects';
export const initSearchProject = (pickerId: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(receiveTreePickerData({
            id: '',
            pickerId,
            data: [{ uuid: SEARCH_PROJECT_ID, name: SEARCH_PROJECT_ID }],
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
    includeDirectories?: boolean;
    includeFiles?: boolean;
    options?: { showOnlyOwned: boolean, showOnlyWritable: boolean };
}

export const loadFavoritesProject = (params: typeof treePickerSearchSagas._Record.LOAD_FAVORITES_PROJECT) => (treePickerSearchSagas.LOAD_FAVORITES_PROJECT(params));

export function* loadFavoritesProjectWatcher() {
    yield takeEvery(treePickerSearchSagas.tags.LOAD_FAVORITES_PROJECT, loadFavoritesProjectSaga);
}

function* loadFavoritesProjectSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.LOAD_FAVORITES_PROJECT,
    payload: typeof treePickerSearchSagas._Record.LOAD_FAVORITES_PROJECT,
}) {
    try {
        const services: ServiceRepository = yield getContext("services");
        const state: RootState = yield select();

        const {
            pickerId,
            includeCollections = false,
            includeDirectories = false,
            includeFiles = false,
            options = { showOnlyOwned: true, showOnlyWritable: false },
        } = payload;
        const uuid = getUserUuid(state);
        if (uuid) {
            const filters = pipe(
                (fb: FilterBuilder) => includeCollections
                    ? fb.addIsA('head_uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION])
                    : fb.addIsA('head_uuid', [ResourceKind.PROJECT]),
                fb => fb.getFilters(),
            )(new FilterBuilder());

            const { items } = yield call(
                {context: services.favoriteService, fn: services.favoriteService.list},
                uuid,
                { filters },
                options.showOnlyOwned,
            );

            yield put(receiveTreePickerData<GroupContentsResource>({
                id: 'Favorites',
                pickerId,
                data: items.filter((item) => {
                    if (options.showOnlyWritable && !(item as GroupResource).canWrite) {
                        return false;
                    }

                    if (options.showOnlyWritable && item.hasOwnProperty('frozenByUuid') && (item as ProjectResource).frozenByUuid) {
                        return false;
                    }

                    return true;
                }),
                extractNodeData: extractGroupContentsNodeData(includeDirectories || includeFiles),
            }));
        }
    } catch(e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to load favorites`, kind: SnackbarKind.ERROR }));
    }
}

export const loadPublicFavoritesProject = (params: typeof treePickerSearchSagas._Record.LOAD_PUBLIC_FAVORITES_PROJECT) => (treePickerSearchSagas.LOAD_PUBLIC_FAVORITES_PROJECT(params));

export function* loadPublicFavoritesProjectWatcher() {
    yield takeEvery(treePickerSearchSagas.tags.LOAD_PUBLIC_FAVORITES_PROJECT, loadPublicFavoritesProjectSaga);
}

function* loadPublicFavoritesProjectSaga({type, payload}: {
    type: typeof treePickerSearchSagas.tags.LOAD_PUBLIC_FAVORITES_PROJECT,
    payload: typeof treePickerSearchSagas._Record.LOAD_PUBLIC_FAVORITES_PROJECT,
}) {
    try {
        const services: ServiceRepository = yield getContext("services");
        const state: RootState = yield select();

        const { pickerId, includeCollections = false, includeDirectories = false, includeFiles = false, options } = payload;
        const uuidPrefix = state.auth.config.uuidPrefix;
        const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;

        // TODO:
        // favorites and public favorites ought to use a single method
        // after getting back a list of links, need to look and stash the resources

        const filters = pipe(
            (fb: FilterBuilder) => includeCollections
                ? fb.addIsA('head_uuid', [ResourceKind.PROJECT, ResourceKind.COLLECTION])
                : fb.addIsA('head_uuid', [ResourceKind.PROJECT]),
            fb => fb
                .addEqual('link_class', LinkClass.STAR)
                .addEqual('owner_uuid', publicProjectUuid)
                .getFilters(),
        )(new FilterBuilder());

        const { items } = yield call(
            {context: services.linkService, fn: services.linkService.list},
            { filters },
        );

        yield put(receiveTreePickerData<LinkResource>({
            id: 'Public Favorites',
            pickerId,
            data: items.filter(item => {
                if (options && options.showOnlyWritable && item.hasOwnProperty('frozenByUuid') && (item as any).frozenByUuid) {
                    return false;
                }

                return true;
            }),
            extractNodeData: item => ({
                id: item.headUuid,
                value: item,
                status: item.headKind === ResourceKind.PROJECT
                    ? TreeNodeStatus.INITIAL
                    : includeDirectories || includeFiles
                        ? TreeNodeStatus.INITIAL
                        : TreeNodeStatus.LOADED
            }),
        }));
    } catch (e) {
        yield put(snackbarActions.OPEN_SNACKBAR({ message: `Failed to load public favorites`, kind: SnackbarKind.ERROR }));
    }
}

export const receiveTreePickerProjectsData = (id: string, projects: ProjectResource[], pickerId: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(treePickerActions.LOAD_TREE_PICKER_NODE_SUCCESS({
            id,
            nodes: projects.map(project => initTreeNode({ id: project.uuid, value: project })),
            pickerId,
        }));

        dispatch(treePickerActions.EXPAND_TREE_PICKER_NODE({ id, pickerId }));
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

/**
 * Given a tree picker item, return collection uuid and path
 *   if the item represents a valid target/destination location
 */
export type FileOperationLocation = {
    name: string;
    uuid: string;
    pdh?: string;
    subpath: string;
}
export const getFileOperationLocation = (item: ProjectsTreePickerItem) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<FileOperationLocation | undefined> => {
        if ('kind' in item && item.kind === ResourceKind.COLLECTION) {
            return {
                name: item.name,
                uuid: item.uuid,
                pdh: item.portableDataHash,
                subpath: '/',
            };
        } else if ('type' in item && item.type === CollectionFileType.DIRECTORY) {
            const uuid = getCollectionResourceCollectionUuid(item.id);
            if (uuid) {
                const collection = getResource<CollectionResource>(uuid)(getState().resources);
                if (collection) {
                    const itemPath = [item.path, item.name].join('/');

                    return {
                        name: item.name,
                        uuid,
                        pdh: collection.portableDataHash,
                        subpath: itemPath,
                    };
                }
            }
        }
        return undefined;
    };

/**
 * Create an expanded tree picker subtree from array of nested projects/collection
 *   First item is assumed to be root and gets empty parent id
 *   Nodes must be sorted from top down to prevent orphaned nodes
 */
export const createInitialPickerTree = (sortedAncestors: Array<GroupResource | CollectionResource>, tailUuid: string, initialTree: Tree<GroupResource | CollectionResource>) => {
    return sortedAncestors
        .reduce((tree, item, index) => {
            if (getNode(item.uuid)(tree)) {
                return tree;
            } else {
                return setNode({
                    children: [],
                    id: item.uuid,
                    parent: index === 0 ? '' : item.ownerUuid,
                    value: item,
                    active: false,
                    selected: false,
                    expanded: false,
                    status: item.uuid !== tailUuid ? TreeNodeStatus.LOADED : TreeNodeStatus.INITIAL,
                })(tree);
            }
        }, initialTree);
};

export const fileOperationLocationToPickerId = (location: FileOperationLocation): string => {
    let id = location.uuid;
    if (location.subpath.length && location.subpath !== '/') {
        id = id + location.subpath;
    }
    return id;
}
