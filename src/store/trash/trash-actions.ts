// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";
import { trashPanelActions } from "~/store/trash-panel/trash-panel-action";
import { activateSidePanelTreeItem, loadSidePanelTreeProjects } from "~/store/side-panel-tree/side-panel-tree-actions";
import { getProjectPanelCurrentUuid, projectPanelActions } from "~/store/project-panel/project-panel-action";
import { ResourceKind } from "~/models/resource";
import { navigateTo, navigateToTrash } from '~/store/navigation/navigation-action';
import { matchCollectionRoute } from '~/routes/routes';

export const toggleProjectTrashed = (uuid: string, ownerUuid: string, isTrashed: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        let errorMessage = '';
        let successMessage = '';
        try {
            if (isTrashed) {
                errorMessage = "Could not restore project from trash";
                successMessage = "Restored from trash";
                await services.groupsService.untrash(uuid);
                dispatch<any>(navigateTo(uuid));
                dispatch<any>(activateSidePanelTreeItem(uuid));
            } else {
                errorMessage = "Could not move project to trash";
                successMessage = "Added to trash";
                await services.groupsService.trash(uuid);
                if (getProjectPanelCurrentUuid(getState()) === uuid) {
                    dispatch<any>(navigateTo(ownerUuid));
                } else {
                    dispatch(projectPanelActions.REQUEST_ITEMS());
                }
                dispatch<any>(loadSidePanelTreeProjects(ownerUuid));
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: errorMessage,
                kind: SnackbarKind.ERROR
            }));
        }
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: successMessage,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS
        }));
    };

export const toggleCollectionTrashed = (uuid: string, isTrashed: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        let errorMessage = '';
        let successMessage = '';
        try {
            if (isTrashed) {
                const { location } = getState().router;
                errorMessage = "Could not restore collection from trash";
                successMessage = "Restored from trash";
                await services.collectionService.untrash(uuid);
                if (matchCollectionRoute(location ? location.pathname : '')) {
                    dispatch(navigateToTrash);
                }
                dispatch(trashPanelActions.REQUEST_ITEMS());
            } else {
                errorMessage = "Could not move collection to trash";
                successMessage = "Added to trash";
                await services.collectionService.trash(uuid);
                dispatch(projectPanelActions.REQUEST_ITEMS());
            }
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: errorMessage,
                kind: SnackbarKind.ERROR
            }));
        }
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: successMessage,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS
        }));
    };

export const toggleTrashed = (kind: ResourceKind, uuid: string, ownerUuid: string, isTrashed: boolean) =>
    (dispatch: Dispatch) => {
        if (kind === ResourceKind.PROJECT) {
            dispatch<any>(toggleProjectTrashed(uuid, ownerUuid, isTrashed!!));
        } else if (kind === ResourceKind.COLLECTION) {
            dispatch<any>(toggleCollectionTrashed(uuid, isTrashed!!));
        }
    };
