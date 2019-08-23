// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from "~/store/store";
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { favoritePanelActions, loadFavoritePanel } from '~/store/favorite-panel/favorite-panel-action';
import {
    getProjectPanelCurrentUuid,
    openProjectPanel,
    projectPanelActions,
    setIsProjectPanelTrashed
} from '~/store/project-panel/project-panel-action';
import {
    activateSidePanelTreeItem,
    initSidePanelTree,
    loadSidePanelTreeProjects,
    SidePanelTreeCategory
} from '~/store/side-panel-tree/side-panel-tree-actions';
import { updateResources } from '~/store/resources/resources-actions';
import { projectPanelColumns } from '~/views/project-panel/project-panel';
import { favoritePanelColumns } from '~/views/favorite-panel/favorite-panel';
import { matchRootRoute } from '~/routes/routes';
import {
    setBreadcrumbs,
    setGroupDetailsBreadcrumbs,
    setGroupsBreadcrumbs,
    setProcessBreadcrumbs,
    setSharedWithMeBreadcrumbs,
    setSidePanelBreadcrumbs,
    setTrashBreadcrumbs
} from '~/store/breadcrumbs/breadcrumbs-actions';
import { navigateTo } from '~/store/navigation/navigation-action';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { ServiceRepository } from '~/services/services';
import { getResource } from '~/store/resources/resources';
import * as projectCreateActions from '~/store/projects/project-create-actions';
import * as projectMoveActions from '~/store/projects/project-move-actions';
import * as projectUpdateActions from '~/store/projects/project-update-actions';
import * as collectionCreateActions from '~/store/collections/collection-create-actions';
import * as collectionCopyActions from '~/store/collections/collection-copy-actions';
import * as collectionUpdateActions from '~/store/collections/collection-update-actions';
import * as collectionMoveActions from '~/store/collections/collection-move-actions';
import * as processesActions from '~/store/processes/processes-actions';
import * as processMoveActions from '~/store/processes/process-move-actions';
import * as processUpdateActions from '~/store/processes/process-update-actions';
import * as processCopyActions from '~/store/processes/process-copy-actions';
import { trashPanelColumns } from "~/views/trash-panel/trash-panel";
import { loadTrashPanel, trashPanelActions } from "~/store/trash-panel/trash-panel-action";
import { initProcessLogsPanel } from '~/store/process-logs-panel/process-logs-panel-actions';
import { loadProcessPanel } from '~/store/process-panel/process-panel-actions';
import {
    loadSharedWithMePanel,
    sharedWithMePanelActions
} from '~/store/shared-with-me-panel/shared-with-me-panel-actions';
import { CopyFormDialogData } from '~/store/copy-dialog/copy-dialog';
import { loadWorkflowPanel, workflowPanelActions } from '~/store/workflow-panel/workflow-panel-actions';
import { loadSshKeysPanel } from '~/store/auth/auth-action-ssh';
import { loadMyAccountPanel } from '~/store/my-account/my-account-panel-actions';
import { loadLinkAccountPanel, linkAccountPanelActions } from '~/store/link-account-panel/link-account-panel-actions';
import { loadSiteManagerPanel } from '~/store/auth/auth-action-session';
import { workflowPanelColumns } from '~/views/workflow-panel/workflow-panel-view';
import { progressIndicatorActions } from '~/store/progress-indicator/progress-indicator-actions';
import { getProgressIndicator } from '~/store/progress-indicator/progress-indicator-reducer';
import { extractUuidKind, ResourceKind } from '~/models/resource';
import { FilterBuilder } from '~/services/api/filter-builder';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { MatchCases, ofType, unionize, UnionOf } from '~/common/unionize';
import { loadRunProcessPanel } from '~/store/run-process-panel/run-process-panel-actions';
import { collectionPanelActions, loadCollectionPanel } from "~/store/collection-panel/collection-panel-action";
import { CollectionResource } from "~/models/collection";
import {
    loadSearchResultsPanel,
    searchResultsPanelActions
} from '~/store/search-results-panel/search-results-panel-actions';
import { searchResultsPanelColumns } from '~/views/search-results-panel/search-results-panel-view';
import { loadVirtualMachinesPanel } from '~/store/virtual-machines/virtual-machines-actions';
import { loadRepositoriesPanel } from '~/store/repositories/repositories-actions';
import { loadKeepServicesPanel } from '~/store/keep-services/keep-services-actions';
import { loadUsersPanel, userBindedActions } from '~/store/users/users-actions';
import { linkPanelActions, loadLinkPanel } from '~/store/link-panel/link-panel-actions';
import { computeNodesActions, loadComputeNodesPanel } from '~/store/compute-nodes/compute-nodes-actions';
import { linkPanelColumns } from '~/views/link-panel/link-panel-root';
import { userPanelColumns } from '~/views/user-panel/user-panel';
import { computeNodePanelColumns } from '~/views/compute-node-panel/compute-node-panel-root';
import { loadApiClientAuthorizationsPanel, apiClientAuthorizationsActions } from '~/store/api-client-authorizations/api-client-authorizations-actions';
import { apiClientAuthorizationPanelColumns } from '~/views/api-client-authorization-panel/api-client-authorization-panel-root';
import * as groupPanelActions from '~/store/groups-panel/groups-panel-actions';
import { groupsPanelColumns } from '~/views/groups-panel/groups-panel';
import * as groupDetailsPanelActions from '~/store/group-details-panel/group-details-panel-actions';
import { groupDetailsPanelColumns } from '~/views/group-details-panel/group-details-panel';
import { DataTableFetchMode } from "~/components/data-table/data-table";
import { loadPublicFavoritePanel, publicFavoritePanelActions } from '~/store/public-favorites-panel/public-favorites-action';
import { publicFavoritePanelColumns } from '~/views/public-favorites-panel/public-favorites-panel';
import { loadCollectionsContentAddressPanel, collectionsContentAddressActions } from '~/store/collections-content-address-panel/collections-content-address-panel-actions';
import { collectionContentAddressPanelColumns } from '~/views/collection-content-address-panel/collection-content-address-panel';

export const WORKBENCH_LOADING_SCREEN = 'workbenchLoadingScreen';

export const isWorkbenchLoading = (state: RootState) => {
    const progress = getProgressIndicator(WORKBENCH_LOADING_SCREEN)(state.progressIndicator);
    return progress ? progress.working : false;
};

const handleFirstTimeLoad = (action: any) =>
    async (dispatch: Dispatch<any>, getState: () => RootState) => {
        try {
            await dispatch(action);
        } finally {
            if (isWorkbenchLoading(getState())) {
                dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
            }
        }
    };

export const loadWorkbench = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
        const { auth, router } = getState();
        const { user } = auth;
        if (user) {
            dispatch(projectPanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
            dispatch(favoritePanelActions.SET_COLUMNS({ columns: favoritePanelColumns }));
            dispatch(publicFavoritePanelActions.SET_COLUMNS({ columns: publicFavoritePanelColumns }));
            dispatch(trashPanelActions.SET_COLUMNS({ columns: trashPanelColumns }));
            dispatch(sharedWithMePanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
            dispatch(workflowPanelActions.SET_COLUMNS({ columns: workflowPanelColumns }));
            dispatch(searchResultsPanelActions.SET_FETCH_MODE({ fetchMode: DataTableFetchMode.INFINITE }));
            dispatch(searchResultsPanelActions.SET_COLUMNS({ columns: searchResultsPanelColumns }));
            dispatch(userBindedActions.SET_COLUMNS({ columns: userPanelColumns }));
            dispatch(groupPanelActions.GroupsPanelActions.SET_COLUMNS({ columns: groupsPanelColumns }));
            dispatch(groupDetailsPanelActions.GroupDetailsPanelActions.SET_COLUMNS({ columns: groupDetailsPanelColumns }));
            dispatch(linkPanelActions.SET_COLUMNS({ columns: linkPanelColumns }));
            dispatch(computeNodesActions.SET_COLUMNS({ columns: computeNodePanelColumns }));
            dispatch(apiClientAuthorizationsActions.SET_COLUMNS({ columns: apiClientAuthorizationPanelColumns }));
            dispatch(collectionsContentAddressActions.SET_COLUMNS({ columns: collectionContentAddressPanelColumns }));

            if (services.linkAccountService.getAccountToLink()) {
                dispatch(linkAccountPanelActions.HAS_SESSION_DATA());
            }

            dispatch<any>(initSidePanelTree());
            if (router.location) {
                const match = matchRootRoute(router.location.pathname);
                if (match) {
                    dispatch<any>(navigateTo(user.uuid));
                }
            }
        } else {
            dispatch(userIsNotAuthenticated);
        }
    };

export const loadFavorites = () =>
    handleFirstTimeLoad(
        (dispatch: Dispatch) => {
            dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.FAVORITES));
            dispatch<any>(loadFavoritePanel());
            dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.FAVORITES));
        });

export const loadCollectionContentAddress = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadCollectionsContentAddressPanel());
    });

export const loadTrash = () =>
    handleFirstTimeLoad(
        (dispatch: Dispatch) => {
            dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
            dispatch<any>(loadTrashPanel());
            dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.TRASH));
        });

export const loadProject = (uuid: string) =>
    handleFirstTimeLoad(
        async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
            const userUuid = services.authService.getUuid();
            dispatch(setIsProjectPanelTrashed(false));
            if (userUuid) {
                if (extractUuidKind(uuid) === ResourceKind.USER && userUuid !== uuid) {
                    // Load another users home projects
                    dispatch(finishLoadingProject(uuid));
                } else if (userUuid !== uuid) {
                    const match = await loadGroupContentsResource({ uuid, userUuid, services });
                    match({
                        OWNED: async project => {
                            await dispatch(activateSidePanelTreeItem(uuid));
                            dispatch<any>(setSidePanelBreadcrumbs(uuid));
                            dispatch(finishLoadingProject(project));
                        },
                        SHARED: project => {
                            dispatch<any>(setSharedWithMeBreadcrumbs(uuid));
                            dispatch(activateSidePanelTreeItem(uuid));
                            dispatch(finishLoadingProject(project));
                        },
                        TRASHED: project => {
                            dispatch<any>(setTrashBreadcrumbs(uuid));
                            dispatch(setIsProjectPanelTrashed(true));
                            dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
                            dispatch(finishLoadingProject(project));
                        }
                    });
                } else {
                    await dispatch(activateSidePanelTreeItem(userUuid));
                    dispatch<any>(setSidePanelBreadcrumbs(userUuid));
                    dispatch(finishLoadingProject(userUuid));
                }
            }
        });

export const createProject = (data: projectCreateActions.ProjectCreateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const newProject = await dispatch<any>(projectCreateActions.createProject(data));
        if (newProject) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully created.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
            await dispatch<any>(loadSidePanelTreeProjects(newProject.ownerUuid));
            dispatch<any>(reloadProjectMatchingUuid([newProject.ownerUuid]));
        }
    };

export const moveProject = (data: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const oldProject = getResource(data.uuid)(getState().resources);
            const oldOwnerUuid = oldProject ? oldProject.ownerUuid : '';
            const movedProject = await dispatch<any>(projectMoveActions.moveProject(data));
            if (movedProject) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Project has been moved', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
                if (oldProject) {
                    await dispatch<any>(loadSidePanelTreeProjects(oldProject.ownerUuid));
                }
                dispatch<any>(reloadProjectMatchingUuid([oldOwnerUuid, movedProject.ownerUuid, movedProject.uuid]));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const updateProject = (data: projectUpdateActions.ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const updatedProject = await dispatch<any>(projectUpdateActions.updateProject(data));
        if (updatedProject) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully updated.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
            await dispatch<any>(loadSidePanelTreeProjects(updatedProject.ownerUuid));
            dispatch<any>(reloadProjectMatchingUuid([updatedProject.ownerUuid, updatedProject.uuid]));
        }
    };

export const loadCollection = (uuid: string) =>
    handleFirstTimeLoad(
        async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
            const userUuid = services.authService.getUuid();
            if (userUuid) {
                const match = await loadGroupContentsResource({ uuid, userUuid, services });
                match({
                    OWNED: async collection => {
                        dispatch(collectionPanelActions.SET_COLLECTION(collection as CollectionResource));
                        dispatch(updateResources([collection]));
                        await dispatch(activateSidePanelTreeItem(collection.ownerUuid));
                        dispatch(setSidePanelBreadcrumbs(collection.ownerUuid));
                        dispatch(loadCollectionPanel(collection.uuid));
                    },
                    SHARED: collection => {
                        dispatch(collectionPanelActions.SET_COLLECTION(collection as CollectionResource));
                        dispatch(updateResources([collection]));
                        dispatch<any>(setSharedWithMeBreadcrumbs(collection.ownerUuid));
                        dispatch(activateSidePanelTreeItem(collection.ownerUuid));
                        dispatch(loadCollectionPanel(collection.uuid));
                    },
                    TRASHED: collection => {
                        dispatch(collectionPanelActions.SET_COLLECTION(collection as CollectionResource));
                        dispatch(updateResources([collection]));
                        dispatch(setTrashBreadcrumbs(''));
                        dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
                        dispatch(loadCollectionPanel(collection.uuid));
                    },

                });
            }
        });

export const createCollection = (data: collectionCreateActions.CollectionCreateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const collection = await dispatch<any>(collectionCreateActions.createCollection(data));
        if (collection) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully created.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
        }
    };

export const updateCollection = (data: collectionUpdateActions.CollectionUpdateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const collection = await dispatch<any>(collectionUpdateActions.updateCollection(data));
        if (collection) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully updated.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
        }
    };

export const copyCollection = (data: CopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const copyToProject = getResource(data.ownerUuid)(getState().resources);
            const collection = await dispatch<any>(collectionCopyActions.copyCollection(data));
            if (copyToProject && collection) {
                dispatch<any>(reloadProjectMatchingUuid([copyToProject.uuid]));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: 'Collection has been copied.',
                    hideDuration: 3000,
                    kind: SnackbarKind.SUCCESS,
                    link: collection.ownerUuid
                }));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const moveCollection = (data: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const collection = await dispatch<any>(collectionMoveActions.moveCollection(data));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been moved.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const loadProcess = (uuid: string) =>
    handleFirstTimeLoad(
        async (dispatch: Dispatch, getState: () => RootState) => {
            dispatch<any>(loadProcessPanel(uuid));
            const process = await dispatch<any>(processesActions.loadProcess(uuid));
            await dispatch<any>(activateSidePanelTreeItem(process.containerRequest.ownerUuid));
            dispatch<any>(setProcessBreadcrumbs(uuid));
            dispatch(loadDetailsPanel(uuid));
        });

export const updateProcess = (data: processUpdateActions.ProcessUpdateFormDialogData) =>
    async (dispatch: Dispatch) => {
        try {
            const process = await dispatch<any>(processUpdateActions.updateProcess(data));
            if (process) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Process has been successfully updated.",
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }));
                dispatch<any>(updateResources([process]));
                dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const moveProcess = (data: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await dispatch<any>(processMoveActions.moveProcess(data));
            dispatch<any>(updateResources([process]));
            dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process has been moved.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const copyProcess = (data: CopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await dispatch<any>(processCopyActions.copyProcess(data));
            dispatch<any>(updateResources([process]));
            dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process has been copied.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const loadProcessLog = (uuid: string) =>
    handleFirstTimeLoad(
        async (dispatch: Dispatch) => {
            const process = await dispatch<any>(processesActions.loadProcess(uuid));
            dispatch<any>(setProcessBreadcrumbs(uuid));
            dispatch<any>(initProcessLogsPanel(uuid));
            await dispatch<any>(activateSidePanelTreeItem(process.containerRequest.ownerUuid));
        });

export const resourceIsNotLoaded = (uuid: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Resource identified by ${uuid} is not loaded.`,
        kind: SnackbarKind.ERROR
    });

export const userIsNotAuthenticated = snackbarActions.OPEN_SNACKBAR({
    message: 'User is not authenticated',
    kind: SnackbarKind.ERROR
});

export const couldNotLoadUser = snackbarActions.OPEN_SNACKBAR({
    message: 'Could not load user',
    kind: SnackbarKind.ERROR
});

export const reloadProjectMatchingUuid = (matchingUuids: string[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentProjectPanelUuid = getProjectPanelCurrentUuid(getState());
        if (currentProjectPanelUuid && matchingUuids.some(uuid => uuid === currentProjectPanelUuid)) {
            dispatch<any>(loadProject(currentProjectPanelUuid));
        }
    };

export const loadSharedWithMe = handleFirstTimeLoad(async (dispatch: Dispatch) => {
    dispatch<any>(loadSharedWithMePanel());
    await dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.SHARED_WITH_ME));
    await dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.SHARED_WITH_ME));
});

export const loadRunProcess = handleFirstTimeLoad(
    async (dispatch: Dispatch) => {
        await dispatch<any>(loadRunProcessPanel());
    }
);

export const loadWorkflow = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.WORKFLOWS));
    await dispatch(loadWorkflowPanel());
    dispatch(setSidePanelBreadcrumbs(SidePanelTreeCategory.WORKFLOWS));
});

export const loadPublicFavorites = () =>
    handleFirstTimeLoad(
        (dispatch: Dispatch) => {
            dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.PUBLIC_FAVORITES));
            dispatch<any>(loadPublicFavoritePanel());
            dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.PUBLIC_FAVORITES));
        });

export const loadSearchResults = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadSearchResultsPanel());
    });

export const loadLinks = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadLinkPanel());
    });

export const loadVirtualMachines = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadVirtualMachinesPanel());
        dispatch(setBreadcrumbs([{ label: 'Virtual Machines' }]));
    });

export const loadRepositories = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadRepositoriesPanel());
        dispatch(setBreadcrumbs([{ label: 'Repositories' }]));
    });

export const loadSshKeys = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadSshKeysPanel());
    });

export const loadSiteManager = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadSiteManagerPanel());
    });

export const loadMyAccount = handleFirstTimeLoad(
    (dispatch: Dispatch<any>) => {
        dispatch(loadMyAccountPanel());
    });

export const loadLinkAccount = handleFirstTimeLoad(
    (dispatch: Dispatch<any>) => {
        dispatch(loadLinkAccountPanel());
    });

export const loadKeepServices = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadKeepServicesPanel());
    });

export const loadUsers = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadUsersPanel());
        dispatch(setBreadcrumbs([{ label: 'Users' }]));
    });

export const loadComputeNodes = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadComputeNodesPanel());
    });

export const loadApiClientAuthorizations = handleFirstTimeLoad(
    async (dispatch: Dispatch<any>) => {
        await dispatch(loadApiClientAuthorizationsPanel());
    });

export const loadGroupsPanel = handleFirstTimeLoad(
    (dispatch: Dispatch<any>) => {
        dispatch(setGroupsBreadcrumbs());
        dispatch(groupPanelActions.loadGroupsPanel());
    });


export const loadGroupDetailsPanel = (groupUuid: string) =>
    handleFirstTimeLoad(
        (dispatch: Dispatch<any>) => {
            dispatch(setGroupDetailsBreadcrumbs(groupUuid));
            dispatch(groupDetailsPanelActions.loadGroupDetailsPanel(groupUuid));
        });

const finishLoadingProject = (project: GroupContentsResource | string) =>
    async (dispatch: Dispatch<any>) => {
        const uuid = typeof project === 'string' ? project : project.uuid;
        dispatch(openProjectPanel(uuid));
        dispatch(loadDetailsPanel(uuid));
        if (typeof project !== 'string') {
            dispatch(updateResources([project]));
        }
    };

const loadGroupContentsResource = async (params: {
    uuid: string,
    userUuid: string,
    services: ServiceRepository
}) => {
    const filters = new FilterBuilder()
        .addEqual('uuid', params.uuid)
        .getFilters();
    const { items } = await params.services.groupsService.contents(params.userUuid, {
        filters,
        recursive: true,
        includeTrash: true,
    });
    const resource = items.shift();
    let handler: GroupContentsHandler;
    if (resource) {
        handler = (resource.kind === ResourceKind.COLLECTION || resource.kind === ResourceKind.PROJECT) && resource.isTrashed
            ? groupContentsHandlers.TRASHED(resource)
            : groupContentsHandlers.OWNED(resource);
    } else {
        const kind = extractUuidKind(params.uuid);
        let resource: GroupContentsResource;
        if (kind === ResourceKind.COLLECTION) {
            resource = await params.services.collectionService.get(params.uuid);
        } else if (kind === ResourceKind.PROJECT) {
            resource = await params.services.projectService.get(params.uuid);
        } else {
            resource = await params.services.containerRequestService.get(params.uuid);
        }
        handler = groupContentsHandlers.SHARED(resource);
    }
    return (cases: MatchCases<typeof groupContentsHandlersRecord, GroupContentsHandler, void>) =>
        groupContentsHandlers.match(handler, cases);

};

const groupContentsHandlersRecord = {
    TRASHED: ofType<GroupContentsResource>(),
    SHARED: ofType<GroupContentsResource>(),
    OWNED: ofType<GroupContentsResource>(),
};

const groupContentsHandlers = unionize(groupContentsHandlersRecord);

type GroupContentsHandler = UnionOf<typeof groupContentsHandlers>;
