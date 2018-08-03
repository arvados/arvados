// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";

import { RootState } from "../../store";
import { ServiceRepository } from "../../../services/services";
import { CollectionResource } from '../../../models/collection';
import { initialize } from 'redux-form';
import { collectionPanelActions } from "../../collection-panel/collection-panel-action";

export const collectionUpdatorActions = unionize({
    OPEN_COLLECTION_UPDATOR: ofType<{ uuid: string }>(),
    CLOSE_COLLECTION_UPDATOR: ofType<{}>(),
    UPDATE_COLLECTION_SUCCESS: ofType<{}>(),
}, {
        tag: 'type',
        value: 'payload'
    });


export const COLLECTION_FORM_NAME = 'collectionEditDialog';
    
export const openUpdator = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(collectionUpdatorActions.OPEN_COLLECTION_UPDATOR({ uuid }));
        const item = getState().collectionPanel.item;
        if(item) {
            dispatch(initialize(COLLECTION_FORM_NAME, { name: item.name, description: item.description }));
        }
    };

export const updateCollection = (collection: Partial<CollectionResource>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { uuid } = getState().collections.updator;
        return services.collectionService
            .update(uuid, collection)
            .then(collection => {
                    dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: collection as CollectionResource }));
                    dispatch(collectionUpdatorActions.UPDATE_COLLECTION_SUCCESS());
                }
            );
    };

export type CollectionUpdatorAction = UnionOf<typeof collectionUpdatorActions>;