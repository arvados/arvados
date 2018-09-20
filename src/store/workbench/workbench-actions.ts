// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from "../store";
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { loadCollectionPanel } from '~/store/collection-panel/collection-panel-action';
import { snackbarActions } from '../snackbar/snackbar-actions';
import { loadFavoritePanel } from '../favorite-panel/favorite-panel-action';
import { openProjectPanel, projectPanelActions } from '~/store/project-panel/project-panel-action';
import { activateSidePanelTreeItem, initSidePanelTree, SidePanelTreeCategory, loadSidePanelTreeProjects, getSidePanelTreeNodeAncestorsIds } from '../side-panel-tree/side-panel-tree-actions';
import { loadResource, updateResources } from '../resources/resources-actions';
import { favoritePanelActions } from '~/store/favorite-panel/favorite-panel-action';
import { projectPanelColumns } from '~/views/project-panel/project-panel';
import { favoritePanelColumns } from '~/views/favorite-panel/favorite-panel';
import { matchRootRoute } from '~/routes/routes';
import { setCollectionBreadcrumbs, setProjectBreadcrumbs, setSidePanelBreadcrumbs, setProcessBreadcrumbs, setSharedWithMeBreadcrumbs } from '../breadcrumbs/breadcrumbs-actions';
import { navigateToProject } from '../navigation/navigation-action';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { ServiceRepository } from '~/services/services';
import { getResource } from '../resources/resources';
import { getProjectPanelCurrentUuid } from '../project-panel/project-panel-action';
import * as projectCreateActions from '~/store/projects/project-create-actions';
import * as projectMoveActions from '~/store/projects/project-move-actions';
import * as projectUpdateActions from '~/store/projects/project-update-actions';
import * as collectionCreateActions from '~/store/collections/collection-create-actions';
import * as collectionCopyActions from '~/store/collections/collection-copy-actions';
import * as collectionUpdateActions from '~/store/collections/collection-update-actions';
import * as collectionMoveActions from '~/store/collections/collection-move-actions';
import * as processesActions from '../processes/processes-actions';
import * as processMoveActions from '~/store/processes/process-move-actions';
import * as processUpdateActions from '~/store/processes/process-update-actions';
import * as processCopyActions from '~/store/processes/process-copy-actions';
import { trashPanelColumns } from "~/views/trash-panel/trash-panel";
import { loadTrashPanel, trashPanelActions } from "~/store/trash-panel/trash-panel-action";
import { initProcessLogsPanel } from '../process-logs-panel/process-logs-panel-actions';
import { loadProcessPanel } from '~/store/process-panel/process-panel-actions';
import { sharedWithMePanelActions } from '~/store/shared-with-me-panel/shared-with-me-panel-actions';
import { loadSharedWithMePanel } from '../shared-with-me-panel/shared-with-me-panel-actions';
import { CopyFormDialogData } from '~/store/copy-dialog/copy-dialog';
import { loadWorkflowPanel, workflowPanelActions } from '~/store/workflow-panel/workflow-panel-actions';
import { workflowPanelColumns } from '~/views/workflow-panel/workflow-panel';

export const loadWorkbench = () =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const { auth, router } = getState();
        const { user } = auth;
        if (user) {
            const userResource = await dispatch<any>(loadResource(user.uuid));
            if (userResource) {
                dispatch(projectPanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
                dispatch(favoritePanelActions.SET_COLUMNS({ columns: favoritePanelColumns }));
                dispatch(trashPanelActions.SET_COLUMNS({ columns: trashPanelColumns }));
                dispatch(sharedWithMePanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
                dispatch(workflowPanelActions.SET_COLUMNS({ columns: workflowPanelColumns}));
                dispatch<any>(initSidePanelTree());
                if (router.location) {
                    const match = matchRootRoute(router.location.pathname);
                    if (match) {
                        dispatch(navigateToProject(userResource.uuid));
                    }
                }
            } else {
                dispatch(userIsNotAuthenticated);
            }
        } else {
            dispatch(userIsNotAuthenticated);
        }
    };

export const loadFavorites = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.FAVORITES));
        dispatch<any>(loadFavoritePanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.FAVORITES));
    };

export const loadTrash = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.TRASH));
        dispatch<any>(loadTrashPanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.TRASH));
    };

export const loadProject = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        dispatch(openProjectPanel(uuid));
        await dispatch(activateSidePanelTreeItem(uuid));
        dispatch(setProjectBreadcrumbs(uuid));
        dispatch(loadDetailsPanel(uuid));
    };

export const createProject = (data: projectCreateActions.ProjectCreateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const newProject = await dispatch<any>(projectCreateActions.createProject(data));
        if (newProject) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully created.",
                hideDuration: 2000
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
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Project has been moved', hideDuration: 2000 }));
                if (oldProject) {
                    await dispatch<any>(loadSidePanelTreeProjects(oldProject.ownerUuid));
                }
                dispatch<any>(reloadProjectMatchingUuid([oldOwnerUuid, movedProject.ownerUuid, movedProject.uuid]));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const updateProject = (data: projectUpdateActions.ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const updatedProject = await dispatch<any>(projectUpdateActions.updateProject(data));
        if (updatedProject) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully updated.",
                hideDuration: 2000
            }));
            await dispatch<any>(loadSidePanelTreeProjects(updatedProject.ownerUuid));
            dispatch<any>(reloadProjectMatchingUuid([updatedProject.ownerUuid, updatedProject.uuid]));
        }
    };

export const loadCollection = (uuid: string) =>
    async (dispatch: Dispatch) => {
        const collection = await dispatch<any>(loadCollectionPanel(uuid));
        await dispatch<any>(activateSidePanelTreeItem(collection.ownerUuid));
        dispatch<any>(setCollectionBreadcrumbs(collection.uuid));
        dispatch(loadDetailsPanel(uuid));
    };

export const createCollection = (data: collectionCreateActions.CollectionCreateFormDialogData) =>
    async (dispatch: Dispatch) => {
        const collection = await dispatch<any>(collectionCreateActions.createCollection(data));
        if (collection) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully created.",
                hideDuration: 2000
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
                hideDuration: 2000
            }));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
        }
    };

export const copyCollection = (data: CopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const collection = await dispatch<any>(collectionCopyActions.copyCollection(data));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been copied.', hideDuration: 2000 }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const moveCollection = (data: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const collection = await dispatch<any>(collectionMoveActions.moveCollection(data));
            dispatch<any>(updateResources([collection]));
            dispatch<any>(reloadProjectMatchingUuid([collection.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Collection has been moved.', hideDuration: 2000 }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const loadProcess = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        dispatch<any>(loadProcessPanel(uuid));
        const process = await dispatch<any>(processesActions.loadProcess(uuid));
        await dispatch<any>(activateSidePanelTreeItem(process.containerRequest.ownerUuid));
        dispatch<any>(setProcessBreadcrumbs(uuid));
        dispatch(loadDetailsPanel(uuid));

    };

export const updateProcess = (data: processUpdateActions.ProcessUpdateFormDialogData) =>
    async (dispatch: Dispatch) => {
        try {
            const process = await dispatch<any>(processUpdateActions.updateProcess(data));
            if (process) {
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Process has been successfully updated.",
                    hideDuration: 2000
                }));
                dispatch<any>(updateResources([process]));
                dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const moveProcess = (data: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await dispatch<any>(processMoveActions.moveProcess(data));
            dispatch<any>(updateResources([process]));
            dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process has been moved.', hideDuration: 2000 }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const copyProcess = (data: CopyFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const process = await dispatch<any>(processCopyActions.copyProcess(data));
            dispatch<any>(updateResources([process]));
            dispatch<any>(reloadProjectMatchingUuid([process.ownerUuid]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Process has been copied.', hideDuration: 2000 }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.message, hideDuration: 2000 }));
        }
    };

export const loadProcessLog = (uuid: string) =>
    async (dispatch: Dispatch) => {
        const process = await dispatch<any>(processesActions.loadProcess(uuid));
        dispatch<any>(setProcessBreadcrumbs(uuid));
        dispatch<any>(initProcessLogsPanel(uuid));
        await dispatch<any>(activateSidePanelTreeItem(process.containerRequest.ownerUuid));
    };

export const resourceIsNotLoaded = (uuid: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Resource identified by ${uuid} is not loaded.`
    });

export const userIsNotAuthenticated = snackbarActions.OPEN_SNACKBAR({
    message: 'User is not authenticated'
});

export const couldNotLoadUser = snackbarActions.OPEN_SNACKBAR({
    message: 'Could not load user'
});

export const reloadProjectMatchingUuid = (matchingUuids: string[]) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const currentProjectPanelUuid = getProjectPanelCurrentUuid(getState());
        if (currentProjectPanelUuid && matchingUuids.some(uuid => uuid === currentProjectPanelUuid)) {
            dispatch<any>(loadProject(currentProjectPanelUuid));
        }
    };

export const loadSharedWithMe = (dispatch: Dispatch) => {
    dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.SHARED_WITH_ME));
    dispatch<any>(loadSharedWithMePanel());
    dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.SHARED_WITH_ME));
};

export const loadWorkflow = (dispatch: Dispatch<any>) => {
    dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.WORKFLOWS));
    dispatch(loadWorkflowPanel());
    dispatch(setSidePanelBreadcrumbs(SidePanelTreeCategory.WORKFLOWS));
};
