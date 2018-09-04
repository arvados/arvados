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
import { ResourceKind, TrashableResource } from "~/models/resource";

export const toggleProjectTrashed = (uuid: string, ownerUuid: string, isTrashed: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        if (isTrashed) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Restoring from trash..." }));
            await services.groupsService.untrash(uuid);
            dispatch<any>(activateSidePanelTreeItem(uuid));
            dispatch(trashPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.CLOSE_SNACKBAR());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Restored from trash",
                hideDuration: 2000
            }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            await services.groupsService.trash(uuid);
            dispatch<any>(loadSidePanelTreeProjects(ownerUuid));
            dispatch(snackbarActions.CLOSE_SNACKBAR());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Added to trash",
                hideDuration: 2000
            }));
        }
    };

export const toggleCollectionTrashed = (uuid: string, isTrashed: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        if (isTrashed) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Restoring from trash..." }));
            await services.collectionService.untrash(uuid);
            dispatch(trashPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Restored from trash",
                hideDuration: 2000
            }));
        } else {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Moving to trash..." }));
            await services.collectionService.trash(uuid);
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Added to trash",
                hideDuration: 2000
            }));
        }
    };

export const toggleTrashed = (kind: ResourceKind, uuid: string, ownerUuid: string, isTrashed: boolean) =>
    (dispatch: Dispatch) => {
        if (kind === ResourceKind.PROJECT) {
            dispatch<any>(toggleProjectTrashed(uuid, ownerUuid, isTrashed!!));
        } else if (kind === ResourceKind.COLLECTION) {
            dispatch<any>(toggleCollectionTrashed(uuid, isTrashed!!));
        }
    };
