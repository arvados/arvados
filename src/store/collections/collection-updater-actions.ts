// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { CollectionResource } from '~/models/collection';
import { initialize } from 'redux-form';
import { collectionPanelActions } from "../collection-panel/collection-panel-action";
import { updateDetails } from "~/store/details-panel/details-panel-action";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { dataExplorerActions } from "~/store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "~/views/project-panel/project-panel";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";

export interface CollectionUpdateFormDialogData {
    name: string;
    description: string;
}

export const COLLECTION_FORM_NAME = 'collectionEditDialog';

export const openUpdater = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(COLLECTION_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_FORM_NAME, data: resource }));
    };

export const editCollection = (data: { name: string, description: string }) =>
    (dispatch: Dispatch) => {
        return dispatch<any>(updateCollection(data)).then(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully updated.",
                hideDuration: 2000
            }));
            dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
        });
    };

export const updateCollection = (collection: Partial<CollectionResource>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = collection.uuid || '';
        return services.collectionService
            .update(uuid, collection)
            .then(collection => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: collection as CollectionResource }));
                dispatch<any>(updateDetails(collection));
                dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_FORM_NAME }));
            }
        );
    };