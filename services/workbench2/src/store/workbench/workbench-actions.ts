// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { getUserUuid } from "common/getuser";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { favoritePanelActions, loadFavoritePanel } from "store/favorite-panel/favorite-panel-action";
import { getProjectPanelCurrentUuid, setIsProjectPanelTrashed } from "store/project-panel/project-panel-action";
import { projectPanelActions } from "store/project-panel/project-panel-action-bind";
import {
    activateSidePanelTreeItem,
    initSidePanelTree,
    loadSidePanelTreeProjects,
    SidePanelTreeCategory,
    SIDE_PANEL_TREE, 
} from "store/side-panel-tree/side-panel-tree-actions";
import { updateResources } from "store/resources/resources-actions";
import { projectPanelColumns } from "views/project-panel/project-panel";
import { favoritePanelColumns } from "views/favorite-panel/favorite-panel";
import { matchRootRoute } from "routes/routes";
import {
    setGroupDetailsBreadcrumbs,
    setGroupsBreadcrumbs,
    setProcessBreadcrumbs,
    setSharedWithMeBreadcrumbs,
    setSidePanelBreadcrumbs,
    setTrashBreadcrumbs,
    setUsersBreadcrumbs,
    setMyAccountBreadcrumbs,
    setUserProfileBreadcrumbs,
    setInstanceTypesBreadcrumbs,
    setVirtualMachinesBreadcrumbs,
    setVirtualMachinesAdminBreadcrumbs,
    setRepositoriesBreadcrumbs,
} from "store/breadcrumbs/breadcrumbs-actions";
import { navigateTo, navigateToRootProject } from "store/navigation/navigation-action";
import { MoveToFormDialogData } from "store/move-to-dialog/move-to-dialog";
import { ServiceRepository } from "services/services";
import { getResource } from "store/resources/resources";
import * as projectCreateActions from "store/projects/project-create-actions";
import * as projectMoveActions from "store/projects/project-move-actions";
import * as projectUpdateActions from "store/projects/project-update-actions";
import * as collectionCreateActions from "store/collections/collection-create-actions";
import * as collectionCopyActions from "store/collections/collection-copy-actions";
import * as collectionMoveActions from "store/collections/collection-move-actions";
import * as processesActions from "store/processes/processes-actions";
import * as processMoveActions from "store/processes/process-move-actions";
import * as processUpdateActions from "store/processes/process-update-actions";
import * as processCopyActions from "store/processes/process-copy-actions";
import { trashPanelColumns } from "views/trash-panel/trash-panel";
import { loadTrashPanel, trashPanelActions } from "store/trash-panel/trash-panel-action";
import { loadProcessPanel } from "store/process-panel/process-panel-actions";
import { loadSharedWithMePanel, sharedWithMePanelActions } from "store/shared-with-me-panel/shared-with-me-panel-actions";
import { sharedWithMePanelColumns } from "views/shared-with-me-panel/shared-with-me-panel";
import { CopyFormDialogData } from "store/copy-dialog/copy-dialog";
import { workflowPanelActions } from "store/workflow-panel/workflow-panel-actions";
import { loadSshKeysPanel } from "store/auth/auth-action-ssh";
import { loadLinkAccountPanel, linkAccountPanelActions } from "store/link-account-panel/link-account-panel-actions";
import { loadSiteManagerPanel } from "store/auth/auth-action-session";
import { workflowPanelColumns } from "views/workflow-panel/workflow-panel-view";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { getProgressIndicator } from "store/progress-indicator/progress-indicator-reducer";
import { extractUuidKind, Resource, ResourceKind } from "models/resource";
import { FilterBuilder } from "services/api/filter-builder";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { MatchCases, ofType, unionize, UnionOf } from "common/unionize";
import { loadRunProcessPanel } from "store/run-process-panel/run-process-panel-actions";
import { collectionPanelActions, loadCollectionPanel } from "store/collection-panel/collection-panel-action";
import { CollectionResource } from "models/collection";
import { WorkflowResource } from "models/workflow";
import { loadSearchResultsPanel, searchResultsPanelActions } from "store/search-results-panel/search-results-panel-actions";
import { searchResultsPanelColumns } from "views/search-results-panel/search-results-panel-view";
import { loadVirtualMachinesPanel } from "store/virtual-machines/virtual-machines-actions";
import { loadRepositoriesPanel } from "store/repositories/repositories-actions";
import { loadKeepServicesPanel } from "store/keep-services/keep-services-actions";
import { loadUsersPanel, userBindedActions } from "store/users/users-actions";
import * as userProfilePanelActions from "store/user-profile/user-profile-actions";
import { linkPanelActions, loadLinkPanel } from "store/link-panel/link-panel-actions";
import { linkPanelColumns } from "views/link-panel/link-panel-root";
import { userPanelColumns } from "views/user-panel/user-panel";
import { loadApiClientAuthorizationsPanel, apiClientAuthorizationsActions } from "store/api-client-authorizations/api-client-authorizations-actions";
import { apiClientAuthorizationPanelColumns } from "views/api-client-authorization-panel/api-client-authorization-panel-root";
import * as groupPanelActions from "store/groups-panel/groups-panel-actions";
import { groupsPanelColumns } from "views/groups-panel/groups-panel";
import * as groupDetailsPanelActions from "store/group-details-panel/group-details-panel-actions";
import { groupDetailsMembersPanelColumns, groupDetailsPermissionsPanelColumns } from "views/group-details-panel/group-details-panel";
import { DataTableFetchMode } from "components/data-table/data-table";
import { loadPublicFavoritePanel, publicFavoritePanelActions } from "store/public-favorites-panel/public-favorites-action";
import { publicFavoritePanelColumns } from "views/public-favorites-panel/public-favorites-panel";
import {
    loadCollectionsContentAddressPanel,
    collectionsContentAddressActions,
} from "store/collections-content-address-panel/collections-content-address-panel-actions";
import { collectionContentAddressPanelColumns } from "views/collection-content-address-panel/collection-content-address-panel";
import { subprocessPanelActions } from "store/subprocess-panel/subprocess-panel-actions";
import { subprocessPanelColumns } from "views/subprocess-panel/subprocess-panel-root";
import { loadAllProcessesPanel, allProcessesPanelActions } from "../all-processes-panel/all-processes-panel-action";
import { allProcessesPanelColumns } from "views/all-processes-panel/all-processes-panel";
import { userProfileGroupsColumns } from "views/user-profile-panel/user-profile-panel-root";
import { selectedToArray, selectedToKindSet } from "components/multiselect-toolbar/MultiselectToolbar";
import { deselectOne } from "store/multiselect/multiselect-actions";
import { treePickerActions } from "store/tree-picker/tree-picker-actions";

export const WORKBENCH_LOADING_SCREEN = "workbenchLoadingScreen";

export const isWorkbenchLoading = (state: RootState) => {
    const progress = getProgressIndicator(WORKBENCH_LOADING_SCREEN)(state.progressIndicator);
    return progress ? progress.working : false;
};

export const handleFirstTimeLoad = (action: any) => async (dispatch: Dispatch<any>, getState: () => RootState) => {
    try {
        await dispatch(action);
    } catch (e) {
        snackbarActions.OPEN_SNACKBAR({
            message: "Error " + e,
            hideDuration: 8000,
            kind: SnackbarKind.WARNING,
        })
    } finally {
        if (isWorkbenchLoading(getState())) {
            dispatch(progressIndicatorActions.STOP_WORKING(WORKBENCH_LOADING_SCREEN));
        }
    }
};

export const loadWorkbench = () => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    dispatch(progressIndicatorActions.START_WORKING(WORKBENCH_LOADING_SCREEN));
    const { auth, router } = getState();
    const { user } = auth;
    if (user) {
        dispatch(projectPanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
        dispatch(favoritePanelActions.SET_COLUMNS({ columns: favoritePanelColumns }));
        dispatch(
            allProcessesPanelActions.SET_COLUMNS({
                columns: allProcessesPanelColumns,
            })
        );
        dispatch(
            publicFavoritePanelActions.SET_COLUMNS({
                columns: publicFavoritePanelColumns,
            })
        );
        dispatch(trashPanelActions.SET_COLUMNS({ columns: trashPanelColumns }));
        dispatch(sharedWithMePanelActions.SET_COLUMNS({ columns: sharedWithMePanelColumns }));
        dispatch(workflowPanelActions.SET_COLUMNS({ columns: workflowPanelColumns }));
        dispatch(
            searchResultsPanelActions.SET_FETCH_MODE({
                fetchMode: DataTableFetchMode.INFINITE,
            })
        );
        dispatch(
            searchResultsPanelActions.SET_COLUMNS({
                columns: searchResultsPanelColumns,
            })
        );
        dispatch(userBindedActions.SET_COLUMNS({ columns: userPanelColumns }));
        dispatch(
            groupPanelActions.GroupsPanelActions.SET_COLUMNS({
                columns: groupsPanelColumns,
            })
        );
        dispatch(
            groupDetailsPanelActions.GroupMembersPanelActions.SET_COLUMNS({
                columns: groupDetailsMembersPanelColumns,
            })
        );
        dispatch(
            groupDetailsPanelActions.GroupPermissionsPanelActions.SET_COLUMNS({
                columns: groupDetailsPermissionsPanelColumns,
            })
        );
        dispatch(
            userProfilePanelActions.UserProfileGroupsActions.SET_COLUMNS({
                columns: userProfileGroupsColumns,
            })
        );
        dispatch(linkPanelActions.SET_COLUMNS({ columns: linkPanelColumns }));
        dispatch(
            apiClientAuthorizationsActions.SET_COLUMNS({
                columns: apiClientAuthorizationPanelColumns,
            })
        );
        dispatch(
            collectionsContentAddressActions.SET_COLUMNS({
                columns: collectionContentAddressPanelColumns,
            })
        );
        dispatch(subprocessPanelActions.SET_COLUMNS({ columns: subprocessPanelColumns }));

        if (services.linkAccountService.getAccountToLink()) {
            dispatch(linkAccountPanelActions.HAS_SESSION_DATA());
        }

        dispatch<any>(initSidePanelTree());
        if (router.location) {
            const match = matchRootRoute(router.location.pathname);
            if (match) {
                dispatch<any>(navigateToRootProject);
            }
        }
    } else {
        dispatch(userIsNotAuthenticated);
    }
};

export const loadFavorites = () =>
    handleFirstTimeLoad((dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.FAVORITES));
        dispatch<any>(loadFavoritePanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.FAVORITES));
    });

export const loadCollectionContentAddress = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadCollectionsContentAddressPanel());
});

export const loadTrash = () =>
    handleFirstTimeLoad((dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
        dispatch<any>(loadTrashPanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.TRASH));
    });

export const loadAllProcesses = () =>
    handleFirstTimeLoad((dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.ALL_PROCESSES));
        dispatch<any>(loadAllProcessesPanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.ALL_PROCESSES));
    });

export const loadProject = (uuid: string) =>
    handleFirstTimeLoad(async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        dispatch(setIsProjectPanelTrashed(false));
        if (!userUuid) {
            return;
        }
        try {
            dispatch(progressIndicatorActions.START_WORKING(uuid));
            if (extractUuidKind(uuid) === ResourceKind.USER && userUuid !== uuid) {
                // Load another users home projects
                dispatch(finishLoadingProject(uuid));
            } else if (userUuid !== uuid) {
                await dispatch(finishLoadingProject(uuid));
                const match = await loadGroupContentsResource({
                    uuid,
                    userUuid,
                    services,
                });
                match({
                    OWNED: async () => {
                        await dispatch(activateSidePanelTreeItem(uuid));
                        dispatch<any>(setSidePanelBreadcrumbs(uuid));
                    },
                    SHARED: async () => {
                        await dispatch(activateSidePanelTreeItem(uuid));
                        dispatch<any>(setSharedWithMeBreadcrumbs(uuid));
                    },
                    TRASHED: async () => {
                        await dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
                        dispatch<any>(setTrashBreadcrumbs(uuid));
                        dispatch(setIsProjectPanelTrashed(true));
                    },
                });
            } else {
                await dispatch(finishLoadingProject(userUuid));
                await dispatch(activateSidePanelTreeItem(userUuid));
                dispatch<any>(setSidePanelBreadcrumbs(userUuid));
            }
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(uuid));
        }
    });

export const createProject = (data: projectCreateActions.ProjectCreateFormDialogData) => async (dispatch: Dispatch) => {
    const newProject = await dispatch<any>(projectCreateActions.createProject(data));
    if (newProject) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully created.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
        await dispatch<any>(loadSidePanelTreeProjects(newProject.ownerUuid));
        dispatch<any>(navigateTo(newProject.uuid));
    }
};

export const moveProject =
    (data: MoveToFormDialogData, isSecondaryMove = false) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
            const checkedList = getState().multiselect.checkedList;
            const uuidsToMove: string[] = data.fromContextMenu ? [data.uuid] : selectedToArray(checkedList);

            //if no items in checkedlist default to normal context menu behavior
            if (!isSecondaryMove && !uuidsToMove.length) uuidsToMove.push(data.uuid);

            const sourceUuid = getResource(data.uuid)(getState().resources)?.ownerUuid;
            const destinationUuid = data.ownerUuid;

            const projectsToMove: MoveableResource[] = uuidsToMove
                .map(uuid => getResource(uuid)(getState().resources) as MoveableResource)
                .filter(resource => resource.kind === ResourceKind.PROJECT);

            for (const project of projectsToMove) {
                await moveSingleProject(project);
            }

            //omly propagate if this call is the original
            if (!isSecondaryMove) {
                const kindsToMove: Set<string> = selectedToKindSet(checkedList);
                kindsToMove.delete(ResourceKind.PROJECT);

                kindsToMove.forEach(kind => {
                    secondaryMove[kind](data, true)(dispatch, getState, services);
                });
            }

            async function moveSingleProject(project: MoveableResource) {
                try {
                    const oldProject: MoveToFormDialogData = { name: project.name, uuid: project.uuid, ownerUuid: data.ownerUuid };
                    const oldOwnerUuid = oldProject ? oldProject.ownerUuid : "";
                    const movedProject = await dispatch<any>(projectMoveActions.moveProject(oldProject));
                    if (movedProject) {
                        dispatch(
                            snackbarActions.OPEN_SNACKBAR({
                                message: "Project has been moved",
                                hideDuration: 2000,
                                kind: SnackbarKind.SUCCESS,
                            })
                        );
                        await dispatch<any>(reloadProjectMatchingUuid([oldOwnerUuid, movedProject.ownerUuid, movedProject.uuid]));
                    }
                } catch (e) {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: !!(project as any).frozenByUuid ? 'Could not move frozen project.' : e.message,
                            hideDuration: 2000,
                            kind: SnackbarKind.ERROR,
                        })
                    );
                }
            }
            if (sourceUuid) await dispatch<any>(loadSidePanelTreeProjects(sourceUuid));
            await dispatch<any>(loadSidePanelTreeProjects(destinationUuid));
        };

export const updateProject = (data: projectUpdateActions.ProjectUpdateFormDialogData) => async (dispatch: Dispatch) => {
    const updatedProject = await dispatch<any>(projectUpdateActions.updateProject(data));
    if (updatedProject) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully updated.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
        await dispatch<any>(loadSidePanelTreeProjects(updatedProject.ownerUuid));
        dispatch<any>(reloadProjectMatchingUuid([updatedProject.ownerUuid, updatedProject.uuid]));
    }
};

export const updateGroup = (data: projectUpdateActions.ProjectUpdateFormDialogData) => async (dispatch: Dispatch) => {
    const updatedGroup = await dispatch<any>(groupPanelActions.updateGroup(data));
    if (updatedGroup) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Group has been successfully updated.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
        await dispatch<any>(loadSidePanelTreeProjects(updatedGroup.ownerUuid));
        dispatch<any>(reloadProjectMatchingUuid([updatedGroup.ownerUuid, updatedGroup.uuid]));
    }
};

export const loadCollection = (uuid: string) =>
    handleFirstTimeLoad(async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        try {
            dispatch(progressIndicatorActions.START_WORKING(uuid));
            if (userUuid) {
                const match = await loadGroupContentsResource({
                    uuid,
                    userUuid,
                    services,
                });
                let collection: CollectionResource | undefined;
                let breadcrumbfunc:
                    | ((uuid: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => Promise<void>)
                    | undefined;
                let sidepanel: string | undefined;
                match({
                    OWNED: thecollection => {
                        collection = thecollection as CollectionResource;
                        sidepanel = collection.ownerUuid;
                        breadcrumbfunc = setSidePanelBreadcrumbs;
                    },
                    SHARED: thecollection => {
                        collection = thecollection as CollectionResource;
                        sidepanel = collection.ownerUuid;
                        breadcrumbfunc = setSharedWithMeBreadcrumbs;
                    },
                    TRASHED: thecollection => {
                        collection = thecollection as CollectionResource;
                        sidepanel = SidePanelTreeCategory.TRASH;
                        breadcrumbfunc = () => setTrashBreadcrumbs("");
                    },
                });
                if (collection && breadcrumbfunc && sidepanel) {
                    dispatch(updateResources([collection]));
                    await dispatch<any>(finishLoadingProject(collection.ownerUuid));
                    dispatch(collectionPanelActions.SET_COLLECTION(collection));
                    await dispatch(activateSidePanelTreeItem(sidepanel));
                    dispatch(breadcrumbfunc(collection.ownerUuid));
                    dispatch(loadCollectionPanel(collection.uuid));
                }
            }
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(uuid));
        }
    });

export const createCollection = (data: collectionCreateActions.CollectionCreateFormDialogData) => async (dispatch: Dispatch) => {
    const collection = await dispatch<any>(collectionCreateActions.createCollection(data));
    if (collection) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully created.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
        dispatch<any>(updateResources([collection]));
        dispatch<any>(navigateTo(collection.uuid));
    }
};

export const copyCollection = (data: CopyFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const checkedList = getState().multiselect.checkedList;
    const uuidsToCopy: string[] = data.fromContextMenu ? [data.uuid] : selectedToArray(checkedList);

    //if no items in checkedlist && no items passed in, default to normal context menu behavior
    if (!uuidsToCopy.length) uuidsToCopy.push(data.uuid);

    const collectionsToCopy: CollectionCopyResource[] = uuidsToCopy
        .map(uuid => getResource(uuid)(getState().resources) as CollectionCopyResource)
        .filter(resource => resource.kind === ResourceKind.COLLECTION);

    for (const collection of collectionsToCopy) {
        await copySingleCollection({ ...collection, ownerUuid: data.ownerUuid } as CollectionCopyResource);
    }

    async function copySingleCollection(copyToProject: CollectionCopyResource) {
        const newName = data.fromContextMenu || collectionsToCopy.length === 1 ? data.name : `Copy of: ${copyToProject.name}`;
        try {
            const collection = await dispatch<any>(
                collectionCopyActions.copyCollection({
                    ...copyToProject,
                    name: newName,
                    fromContextMenu: collectionsToCopy.length === 1 ? true : data.fromContextMenu,
                })
            );
            if (copyToProject && collection) {
                await dispatch<any>(reloadProjectMatchingUuid([copyToProject.uuid]));
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: "Collection has been copied.",
                        hideDuration: 3000,
                        kind: SnackbarKind.SUCCESS,
                        link: collection.ownerUuid,
                    })
                );
                dispatch<any>(deselectOne(copyToProject.uuid));
            }
        } catch (e) {
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: e.message,
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR,
                })
            );
        }
    }
    dispatch(projectPanelActions.REQUEST_ITEMS());
};

export const moveCollection =
    (data: MoveToFormDialogData, isSecondaryMove = false) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
            const checkedList = getState().multiselect.checkedList;
            const uuidsToMove: string[] = data.fromContextMenu ? [data.uuid] : selectedToArray(checkedList);

            //if no items in checkedlist && no items passed in, default to normal context menu behavior
            if (!isSecondaryMove && !uuidsToMove.length) uuidsToMove.push(data.uuid);

            const collectionsToMove: MoveableResource[] = uuidsToMove
                .map(uuid => getResource(uuid)(getState().resources) as MoveableResource)
                .filter(resource => resource.kind === ResourceKind.COLLECTION);

            for (const collection of collectionsToMove) {
                await moveSingleCollection(collection);
            }

            //omly propagate if this call is the original
            if (!isSecondaryMove) {
                const kindsToMove: Set<string> = selectedToKindSet(checkedList);
                kindsToMove.delete(ResourceKind.COLLECTION);

                kindsToMove.forEach(kind => {
                    secondaryMove[kind](data, true)(dispatch, getState, services);
                });
            }

            async function moveSingleCollection(collection: MoveableResource) {
                try {
                    const oldCollection: MoveToFormDialogData = { name: collection.name, uuid: collection.uuid, ownerUuid: data.ownerUuid };
                    const movedCollection = await dispatch<any>(collectionMoveActions.moveCollection(oldCollection));
                    dispatch<any>(updateResources([movedCollection]));
                    dispatch<any>(reloadProjectMatchingUuid([movedCollection.ownerUuid]));
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: "Collection has been moved.",
                            hideDuration: 2000,
                            kind: SnackbarKind.SUCCESS,
                        })
                    );
                } catch (e) {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: e.message,
                            hideDuration: 2000,
                            kind: SnackbarKind.ERROR,
                        })
                    );
                }
            }
        };

export const loadProcess = (uuid: string) =>
    handleFirstTimeLoad(async (dispatch: Dispatch, getState: () => RootState) => {
        try {
            dispatch(progressIndicatorActions.START_WORKING(uuid));
            dispatch<any>(loadProcessPanel(uuid));
            const process = await dispatch<any>(processesActions.loadProcess(uuid));
            if (process) {
                await dispatch<any>(finishLoadingProject(process.containerRequest.ownerUuid));
                await dispatch<any>(activateSidePanelTreeItem(process.containerRequest.ownerUuid));
                dispatch<any>(setProcessBreadcrumbs(uuid));
                dispatch<any>(loadDetailsPanel(uuid));
            }
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(uuid));
        }
    });

export const loadRegisteredWorkflow = (uuid: string) =>
    handleFirstTimeLoad(async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getUserUuid(getState());
        if (userUuid) {
            const match = await loadGroupContentsResource({
                uuid,
                userUuid,
                services,
            });
            let workflow: WorkflowResource | undefined;
            let breadcrumbfunc:
                | ((uuid: string) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => Promise<void>)
                | undefined;
            match({
                OWNED: async theworkflow => {
                    workflow = theworkflow as WorkflowResource;
                    breadcrumbfunc = setSidePanelBreadcrumbs;
                },
                SHARED: async theworkflow => {
                    workflow = theworkflow as WorkflowResource;
                    breadcrumbfunc = setSharedWithMeBreadcrumbs;
                },
                TRASHED: () => { },
            });
            if (workflow && breadcrumbfunc) {
                dispatch(updateResources([workflow]));
                await dispatch<any>(finishLoadingProject(workflow.ownerUuid));
                await dispatch<any>(activateSidePanelTreeItem(workflow.ownerUuid));
                dispatch<any>(breadcrumbfunc(workflow.ownerUuid));
            }
        }
    });

export const updateProcess = (data: processUpdateActions.ProcessUpdateFormDialogData) => async (dispatch: Dispatch) => {
    try {
        const process = await dispatch<any>(processUpdateActions.updateProcess(data));
        if (process) {
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: "Process has been successfully updated.",
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                })
            );
            dispatch<any>(updateResources([process]));
            dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
        }
    } catch (e) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: e.message,
                hideDuration: 2000,
                kind: SnackbarKind.ERROR,
            })
        );
    }
};

export const moveProcess =
    (data: MoveToFormDialogData, isSecondaryMove = false) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
            const checkedList = getState().multiselect.checkedList;
            const uuidsToMove: string[] = data.fromContextMenu ? [data.uuid] : selectedToArray(checkedList);

            //if no items in checkedlist && no items passed in, default to normal context menu behavior
            if (!isSecondaryMove && !uuidsToMove.length) uuidsToMove.push(data.uuid);

            const processesToMove: MoveableResource[] = uuidsToMove
                .map(uuid => getResource(uuid)(getState().resources) as MoveableResource)
                .filter(resource => resource.kind === ResourceKind.PROCESS);

            for (const process of processesToMove) {
                await moveSingleProcess(process);
            }

            //omly propagate if this call is the original
            if (!isSecondaryMove) {
                const kindsToMove: Set<string> = selectedToKindSet(checkedList);
                kindsToMove.delete(ResourceKind.PROCESS);

                kindsToMove.forEach(kind => {
                    secondaryMove[kind](data, true)(dispatch, getState, services);
                });
            }

            async function moveSingleProcess(process: MoveableResource) {
                try {
                    const oldProcess: MoveToFormDialogData = { name: process.name, uuid: process.uuid, ownerUuid: data.ownerUuid };
                    const movedProcess = await dispatch<any>(processMoveActions.moveProcess(oldProcess));
                    dispatch<any>(updateResources([movedProcess]));
                    dispatch<any>(reloadProjectMatchingUuid([movedProcess.ownerUuid]));
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: "Process has been moved.",
                            hideDuration: 2000,
                            kind: SnackbarKind.SUCCESS,
                        })
                    );
                } catch (e) {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: e.message,
                            hideDuration: 2000,
                            kind: SnackbarKind.ERROR,
                        })
                    );
                }
            }
        };

export const copyProcess = (data: CopyFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const process = await dispatch<any>(processCopyActions.copyProcess(data));
        dispatch<any>(updateResources([process]));
        dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: "Process has been copied.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS,
            })
        );
        dispatch<any>(navigateTo(process.uuid));
    } catch (e) {
        dispatch(
            snackbarActions.OPEN_SNACKBAR({
                message: e.message,
                hideDuration: 2000,
                kind: SnackbarKind.ERROR,
            })
        );
    }
};

export const resourceIsNotLoaded = (uuid: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Resource identified by ${uuid} is not loaded.`,
        kind: SnackbarKind.ERROR,
    });

export const userIsNotAuthenticated = snackbarActions.OPEN_SNACKBAR({
    message: "User is not authenticated",
    kind: SnackbarKind.ERROR,
});

export const couldNotLoadUser = snackbarActions.OPEN_SNACKBAR({
    message: "Could not load user",
    kind: SnackbarKind.ERROR,
});

export const reloadProjectMatchingUuid =
    (matchingUuids: string[]) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
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

export const loadRunProcess = handleFirstTimeLoad(async (dispatch: Dispatch) => {
    await dispatch<any>(loadRunProcessPanel());
});

export const loadPublicFavorites = () =>
    handleFirstTimeLoad((dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.PUBLIC_FAVORITES));
        dispatch<any>(loadPublicFavoritePanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.PUBLIC_FAVORITES));
    });

export const loadSearchResults = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadSearchResultsPanel());
});

export const loadLinks = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadLinkPanel());
});

export const loadVirtualMachines = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadVirtualMachinesPanel());
    dispatch(setVirtualMachinesBreadcrumbs());
    dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.SHELL_ACCESS));
});

export const loadVirtualMachinesAdmin = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadVirtualMachinesPanel());
    dispatch(setVirtualMachinesAdminBreadcrumbs());
    dispatch(treePickerActions.DEACTIVATE_TREE_PICKER_NODE({pickerId: SIDE_PANEL_TREE} ))
});

export const loadRepositories = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadRepositoriesPanel());
    dispatch(setRepositoriesBreadcrumbs());
});

export const loadSshKeys = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadSshKeysPanel());
});

export const loadInstanceTypes = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.INSTANCE_TYPES));
    dispatch(setInstanceTypesBreadcrumbs());
});

export const loadSiteManager = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadSiteManagerPanel());
});

export const loadUserProfile = (userUuid?: string) =>
    handleFirstTimeLoad((dispatch: Dispatch<any>) => {
        if (userUuid) {
            dispatch(setUserProfileBreadcrumbs(userUuid));
            dispatch(userProfilePanelActions.loadUserProfilePanel(userUuid));
        } else {
            dispatch(setMyAccountBreadcrumbs());
            dispatch(userProfilePanelActions.loadUserProfilePanel());
        }
    });

export const loadLinkAccount = handleFirstTimeLoad((dispatch: Dispatch<any>) => {
    dispatch(loadLinkAccountPanel());
});

export const loadKeepServices = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadKeepServicesPanel());
});

export const loadUsers = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadUsersPanel());
    dispatch(setUsersBreadcrumbs());
});

export const loadApiClientAuthorizations = handleFirstTimeLoad(async (dispatch: Dispatch<any>) => {
    await dispatch(loadApiClientAuthorizationsPanel());
});

export const loadGroupsPanel = handleFirstTimeLoad((dispatch: Dispatch<any>) => {
    dispatch(setGroupsBreadcrumbs());
    dispatch(groupPanelActions.loadGroupsPanel());
});

export const loadGroupDetailsPanel = (groupUuid: string) =>
    handleFirstTimeLoad((dispatch: Dispatch<any>) => {
        dispatch(setGroupDetailsBreadcrumbs(groupUuid));
        dispatch(groupDetailsPanelActions.loadGroupDetailsPanel(groupUuid));
    });

const finishLoadingProject = (project: GroupContentsResource | string) => async (dispatch: Dispatch<any>) => {
    const uuid = typeof project === "string" ? project : project.uuid;
    dispatch(loadDetailsPanel(uuid));
    if (typeof project !== "string") {
        dispatch(updateResources([project]));
    }
};

const loadGroupContentsResource = async (params: { uuid: string; userUuid: string; services: ServiceRepository }) => {
    const filters = new FilterBuilder().addEqual("uuid", params.uuid).getFilters();
    const { items } = await params.services.groupsService.contents(params.userUuid, {
        filters,
        recursive: true,
        includeTrash: true,
    });
    const resource = items.shift();
    let handler: GroupContentsHandler;
    if (resource) {
        handler =
            (resource.kind === ResourceKind.COLLECTION || resource.kind === ResourceKind.PROJECT) && resource.isTrashed
                ? groupContentsHandlers.TRASHED(resource)
                : groupContentsHandlers.OWNED(resource);
    } else {
        const kind = extractUuidKind(params.uuid);
        let resource: GroupContentsResource;
        if (kind === ResourceKind.COLLECTION) {
            resource = await params.services.collectionService.get(params.uuid);
        } else if (kind === ResourceKind.PROJECT) {
            resource = await params.services.projectService.get(params.uuid);
        } else if (kind === ResourceKind.WORKFLOW) {
            resource = await params.services.workflowService.get(params.uuid);
        } else if (kind === ResourceKind.CONTAINER_REQUEST) {
            resource = await params.services.containerRequestService.get(params.uuid);
        } else {
            throw new Error("loadGroupContentsResource unsupported kind " + kind);
        }
        handler = groupContentsHandlers.SHARED(resource);
    }
    return (cases: MatchCases<typeof groupContentsHandlersRecord, GroupContentsHandler, void>) => groupContentsHandlers.match(handler, cases);
};

const groupContentsHandlersRecord = {
    TRASHED: ofType<GroupContentsResource>(),
    SHARED: ofType<GroupContentsResource>(),
    OWNED: ofType<GroupContentsResource>(),
};

const groupContentsHandlers = unionize(groupContentsHandlersRecord);

type GroupContentsHandler = UnionOf<typeof groupContentsHandlers>;

type CollectionCopyResource = Resource & { name: string; fromContextMenu: boolean };

type MoveableResource = Resource & { name: string };

type MoveFunc = (
    data: MoveToFormDialogData,
    isSecondaryMove?: boolean
) => (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => Promise<void>;

const secondaryMove: Record<string, MoveFunc> = {
    [ResourceKind.PROJECT]: moveProject,
    [ResourceKind.PROCESS]: moveProcess,
    [ResourceKind.COLLECTION]: moveCollection,
};
