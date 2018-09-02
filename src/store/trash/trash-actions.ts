// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { trashPanelActions } from "~/store/trash-panel/trash-panel-action";
import { activateSidePanelTreeItem, loadSidePanelTreeProjects } from "~/store/side-panel-tree/side-panel-tree-actions";
import { projectPanelActions } from "~/store/project-panel/project-panel-action";

export const toggleProjectTrashed = (resource: { uuid: string; name: string, isTrashed?: boolean, ownerUuid?: string }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        if (resource.isTrashed) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Restoring from trash..." }));
            return services.groupsService.untrash(resource.uuid).then(() => {
                dispatch<any>(activateSidePanelTreeItem(resource.uuid));
                dispatch(trashPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Restored from trash",
                    hideDuration: 2000
                }));
            });
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            return services.groupsService.trash(resource.uuid).then(() => {
                dispatch<any>(loadSidePanelTreeProjects(resource.ownerUuid!!));
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Added to trash",
                    hideDuration: 2000
                }));
            });
        }
    };

export const toggleCollectionTrashed = (resource: { uuid: string; name: string, isTrashed?: boolean, ownerUuid?: string }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        if (resource.isTrashed) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Restoring from trash..." }));
            return services.collectionService.untrash(resource.uuid).then(() => {
                dispatch(trashPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Restored from trash",
                    hideDuration: 2000
                }));
            });
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            return services.collectionService.trash(resource.uuid).then(() => {
                dispatch(projectPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Added to trash",
                    hideDuration: 2000
                }));
            });
        }
    };
