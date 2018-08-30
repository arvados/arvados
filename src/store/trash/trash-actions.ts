// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { trashPanelActions } from "~/store/trash-panel/trash-panel-action";

export const toggleProjectTrashed = (resource: { uuid: string; name: string, isTrashed?: boolean, ownerUuid?: string }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Working..." }));
        if (resource.isTrashed) {
            return services.groupsService.untrash(resource.uuid).then(() => {
                // dispatch<any>(getProjectList(resource.ownerUuid)).then(() => {
                //     dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(SidePanelId.PROJECTS));
                //     dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN({ itemId: resource.ownerUuid!!, open: true, recursive: true }));
                // });
                dispatch(trashPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Restored from trash",
                    hideDuration: 2000
                }));
            });
        } else {
            return services.groupsService.trash(resource.uuid).then(() => {
                // dispatch<any>(getProjectList(resource.ownerUuid)).then(() => {
                //     dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN({ itemId: resource.ownerUuid!!, open: true, recursive: true }));
                // });
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
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Working..." }));
        if (resource.isTrashed) {
            return services.collectionService.untrash(resource.uuid).then(() => {
                // dispatch<any>(getProjectList(resource.ownerUuid)).then(() => {
                //     dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(SidePanelId.PROJECTS));
                //     dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN({ itemId: resource.ownerUuid!!, open: true, recursive: true }));
                // });
                dispatch(trashPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Restored from trash",
                    hideDuration: 2000
                }));
            });
        } else {
            return services.collectionService.trash(resource.uuid).then(() => {
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Added to trash",
                    hideDuration: 2000
                }));
            });
        }
    };
