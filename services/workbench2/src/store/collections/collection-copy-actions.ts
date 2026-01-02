// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { CopyFormDialogData } from "store/copy-dialog/copy-dialog";
import { initProjectsTreePicker } from "store/tree-picker/tree-picker-actions";
import { getResource } from "store/resources/resources";
import { CollectionResource } from "models/collection";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { getResourcesFromCheckedList } from "store/multiselect/multiselect-actions";

export const COLLECTION_COPY_FORM_NAME = "collectionCopyFormName";

export const openCollectionCopy = (resource: { name: string; uuid: string; }) => (dispatch: Dispatch, getState: () => RootState) => {
    const resourcesToCopy = getResourcesFromCheckedList(getState()).filter(res => !!res).map(res => ({ name: res!.name, uuid: res!.uuid }));
    if (!resourcesToCopy.length) resourcesToCopy.push(resource);
    const isSingleResource = resourcesToCopy.length === 1;
    dispatch<any>(resetPickerProjectTree());
    dispatch<any>(initProjectsTreePicker(COLLECTION_COPY_FORM_NAME));
    const initialData: CopyFormDialogData = { name: `Copy of: ${resource.name}`, ownerUuid: "", uuid: resource.uuid, isSingleResource };
    dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_COPY_FORM_NAME, data: initialData }) );
}

export const copyCollection =
    (resource: CopyFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
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
            return newCollection;
        } catch (e) {
            console.error("Error while copying collection: ", e);
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: "A collection with the same name already exists in the target project.",
                        hideDuration: 3000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            } else {
                throw new Error("Could not copy the collection.");
            }
            return;
        }
    };
