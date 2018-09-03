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
            await services.groupsService.untrash(resource.uuid);
            dispatch<any>(activateSidePanelTreeItem(resource.uuid));
            dispatch(trashPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.CLOSE_SNACKBAR());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Restored from trash",
                hideDuration: 2000
            }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            await services.groupsService.trash(resource.uuid);
            dispatch<any>(loadSidePanelTreeProjects(resource.ownerUuid!!));
            dispatch(snackbarActions.CLOSE_SNACKBAR());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Added to trash",
                hideDuration: 2000
            }));
        }
    };

export const toggleCollectionTrashed = (resource: { uuid: string; name: string, isTrashed?: boolean, ownerUuid?: string }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        if (resource.isTrashed) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Restoring from trash..." }));
            await services.collectionService.untrash(resource.uuid);
            dispatch(trashPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Restored from trash",
                hideDuration: 2000
            }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            await services.collectionService.trash(resource.uuid);
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Added to trash",
                hideDuration: 2000
            }));
        }
    };
