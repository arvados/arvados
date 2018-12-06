// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { FormErrors, initialize, startSubmit, stopSubmit } from 'redux-form';
import { RootState } from "~/store/store";
import { collectionPanelActions } from "~/store/collection-panel/collection-panel-action";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import { ServiceRepository } from "~/services/services";
import { CollectionResource } from '~/models/collection';
import { ContextMenuResource } from "~/store/context-menu/context-menu-actions";
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";

export interface CollectionUpdateFormDialogData {
    uuid: string;
    name: string;
    description: string;
}

export const COLLECTION_UPDATE_FORM_NAME = 'collectionUpdateFormName';

export const openCollectionUpdateDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(COLLECTION_UPDATE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: COLLECTION_UPDATE_FORM_NAME, data: {} }));
    };

export const updateCollection = (collection: Partial<CollectionResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = collection.uuid || '';
        dispatch(startSubmit(COLLECTION_UPDATE_FORM_NAME));
        try {
            dispatch(progressIndicatorActions.START_WORKING(COLLECTION_UPDATE_FORM_NAME));
            const updatedCollection = await services.collectionService.update(uuid, collection);
            dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: updatedCollection as CollectionResource }));
            dispatch(dialogActions.CLOSE_DIALOG({ id: COLLECTION_UPDATE_FORM_NAME }));
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_UPDATE_FORM_NAME));
            return updatedCollection;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(COLLECTION_UPDATE_FORM_NAME, { name: 'Collection with the same name already exists.' } as FormErrors));
            }
            dispatch(progressIndicatorActions.STOP_WORKING(COLLECTION_UPDATE_FORM_NAME));
            return;
        }
    };
