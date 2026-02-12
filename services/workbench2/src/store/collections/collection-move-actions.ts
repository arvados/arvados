// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { ServiceRepository } from "services/services";
import { RootState } from "store/store";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { MoveToFormDialogData } from "store/move-to-dialog/move-to-dialog";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { initProjectsTreePicker } from "store/tree-picker/tree-picker-actions";
import { getResource } from "store/resources/resources";
import { CollectionResource } from "models/collection";

export const COLLECTION_MOVE_FORM_NAME = "collectionMoveFormName";

export const openMoveCollectionDialog = (resource: { name: string; uuid: string }) => (dispatch: Dispatch) => {
    dispatch<any>(resetPickerProjectTree());
    dispatch<any>(initProjectsTreePicker(COLLECTION_MOVE_FORM_NAME));
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_MOVE_FORM_NAME, data: resource }));
};

export const moveCollection =
    (resource: MoveToFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        let cachedCollection = getResource<CollectionResource>(resource.uuid)(getState().resources);
        try {
            if (!cachedCollection) {
                cachedCollection = await services.collectionService.get(resource.uuid);
            }
            const collection = await services.collectionService.update(resource.uuid, { ownerUuid: resource.ownerUuid });
            dispatch(projectPanelDataActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_MOVE_FORM_NAME }));
            return { ...cachedCollection, ...collection };
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "A collection with the same name already exists in the target project.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_MOVE_FORM_NAME }));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Could not move the collection.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
            }
            return;
        }
    };
