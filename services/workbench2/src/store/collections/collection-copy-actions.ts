// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { FormErrors, initialize, startSubmit, stopSubmit } from "redux-form";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { CopyFormDialogData } from "store/copy-dialog/copy-dialog";
import { progressIndicatorsActions } from "store/progress-indicator/progress-indicator-actions";
import { initProjectsTreePicker } from "store/tree-picker/tree-picker-actions";
import { getResource } from "store/resources/resources";
import { CollectionResource } from "models/collection";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export const COLLECTION_COPY_FORM_NAME = "collectionCopyFormName";
export const COLLECTION_MULTI_COPY_FORM_NAME = "collectionMultiCopyFormName";

export const openCollectionCopyDialog = (resource: { name: string; uuid: string; fromContextMenu?: boolean }) => (dispatch: Dispatch) => {
    dispatch<any>(resetPickerProjectTree());
    dispatch<any>(initProjectsTreePicker(COLLECTION_COPY_FORM_NAME));
    const initialData: CopyFormDialogData = { name: `Copy of: ${resource.name}`, ownerUuid: "", uuid: resource.uuid, fromContextMenu: resource.fromContextMenu };
    dispatch<any>(initialize(COLLECTION_COPY_FORM_NAME, initialData));
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_COPY_FORM_NAME, data: {} }));
};

export const openMultiCollectionCopyDialog = (resource: { name: string; uuid: string; fromContextMenu?: boolean }) => (dispatch: Dispatch) => {
    dispatch<any>(resetPickerProjectTree());
    dispatch<any>(initProjectsTreePicker(COLLECTION_MULTI_COPY_FORM_NAME));
    const initialData: CopyFormDialogData = { name: `Copy of: ${resource.name}`, ownerUuid: "", uuid: resource.uuid, fromContextMenu: resource.fromContextMenu };
    dispatch<any>(initialize(COLLECTION_MULTI_COPY_FORM_NAME, initialData));
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_MULTI_COPY_FORM_NAME, data: {} }));
};

export const copyCollection =
    (resource: CopyFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const formName = resource.fromContextMenu ? COLLECTION_COPY_FORM_NAME : COLLECTION_MULTI_COPY_FORM_NAME;
        dispatch(startSubmit(formName));
        let collection = getResource<CollectionResource>(resource.uuid)(getState().resources);
        try {
            if (!collection) {
                collection = await services.collectionService.get(resource.uuid);
            }
            const collManifestText = await services.collectionService.get(resource.uuid, undefined, ["manifestText"]);
            collection.manifestText = collManifestText.manifestText;
            const newCollection = await services.collectionService.create(
                {
                    ...collection,
                    ownerUuid: resource.ownerUuid,
                    name: resource.name,
                },
                false
            );
            dispatch(dialogActions.CLOSE_DIALOG({ id: formName }));
            return newCollection;
        } catch (e) {
            console.error("Error while copying collection: ", e);
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(
                    stopSubmit(formName, {
                        ownerUuid: "A collection with the same name already exists in the target project.",
                    } as FormErrors)
                );
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: "Could not copy the collection.",
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: formName }));
                throw new Error("Could not copy the collection.");
            }
            return;
        } finally {
            dispatch(progressIndicatorsActions.STOP_WORKING(formName));
        }
    };
