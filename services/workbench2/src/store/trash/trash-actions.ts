// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { trashPanelActions } from "store/trash-panel/trash-panel-action";
import { activateSidePanelTreeItem, loadSidePanelTreeProjects } from "store/side-panel-tree/side-panel-tree-actions";
import { projectPanelActions } from "store/project-panel/project-panel-action-bind";
import { ResourceKind } from "models/resource";
import { navigateTo, navigateToTrash } from "store/navigation/navigation-action";
import { matchCollectionRoute } from "routes/routes";

export const toggleProjectTrashed =
    (uuid: string, ownerUuid: string, isTrashed: boolean, isMulti: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        let errorMessage = "";
        let successMessage = "";
        let untrashedResource;
        try {
            if (isTrashed) {
                errorMessage = "Could not restore project from trash";
                successMessage = "Restored project from trash";
                untrashedResource = await services.groupsService.untrash(uuid);
                dispatch<any>(isMulti || !untrashedResource ? navigateToTrash : navigateTo(uuid));
                dispatch<any>(activateSidePanelTreeItem(uuid));
            } else {
                errorMessage = "Could not move project to trash";
                successMessage = "Added project to trash";
                await services.groupsService.trash(uuid);
                dispatch<any>(loadSidePanelTreeProjects(ownerUuid));
                dispatch<any>(navigateTo(ownerUuid));
            }
            if (untrashedResource) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: successMessage,
                        hideDuration: 2000,
                        kind: SnackbarKind.SUCCESS,
                    })
                );
            }
        } catch (e) {
            if (e.status === 422) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: "Could not restore project from trash: Duplicate name at destination",
                        kind: SnackbarKind.ERROR,
                    })
                );
            } else {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: errorMessage,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
        }
    };

export const toggleCollectionTrashed =
    (uuid: string, isTrashed: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        let errorMessage = "";
        let successMessage = "";
        try {
            if (isTrashed) {
                const { location } = getState().router;
                errorMessage = "Could not restore collection from trash";
                successMessage = "Restored from trash";
                await services.collectionService.untrash(uuid);
                if (matchCollectionRoute(location ? location.pathname : "")) {
                    dispatch(navigateToTrash);
                }
                dispatch(trashPanelActions.REQUEST_ITEMS());
            } else {
                errorMessage = "Could not move collection to trash";
                successMessage = "Added to trash";
                await services.collectionService.trash(uuid);
                dispatch(projectPanelActions.REQUEST_ITEMS());
            }
            dispatch(
                snackbarActions.OPEN_SNACKBAR({
                    message: successMessage,
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS,
                })
            );
        } catch (e) {
            if (e.status === 422) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: "Could not restore collection from trash: Duplicate name at destination",
                        kind: SnackbarKind.ERROR,
                    })
                );
            } else {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: errorMessage,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
        }
    };

export const toggleTrashed = (kind: ResourceKind, uuid: string, ownerUuid: string, isTrashed: boolean) => (dispatch: Dispatch) => {
    if (kind === ResourceKind.PROJECT) {
        dispatch<any>(toggleProjectTrashed(uuid, ownerUuid, isTrashed!!, false));
    } else if (kind === ResourceKind.COLLECTION) {
        dispatch<any>(toggleCollectionTrashed(uuid, isTrashed!!));
    }
};
