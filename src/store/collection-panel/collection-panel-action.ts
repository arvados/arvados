// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { loadCollectionFiles } from "./collection-panel-files/collection-panel-files-actions";
import { CollectionResource } from '~/models/collection';
import { collectionPanelFilesAction } from "./collection-panel-files/collection-panel-files-actions";
import { createTree } from "~/models/tree";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { TagProperty } from "~/models/tag";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { resourcesActions } from "~/store/resources/resources-actions";
import { unionize, ofType, UnionOf } from '~/common/unionize';

export const collectionPanelActions = unionize({
    SET_COLLECTION: ofType<CollectionResource>(),
    LOAD_COLLECTION: ofType<{ uuid: string }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>()
});

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const COLLECTION_TAG_FORM_NAME = 'collectionTagForm';

export const loadCollectionPanel = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid }));
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ files: createTree() }));
        const collection = await services.collectionService.get(uuid);
        dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: collection }));
        dispatch(resourcesActions.SET_RESOURCES([collection]));
        dispatch<any>(loadCollectionFiles(collection.uuid));
        return collection;
    };

export const createCollectionTag = (data: TagProperty) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const item = getState().collectionPanel.item;
        const uuid = item ? item.uuid : '';
        try {
            if (item) {
                item.properties[data.key] = data.value;
                const updatedCollection = await services.collectionService.update(uuid, item);
                dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Tag has been successfully added.", hideDuration: 2000 }));
                return updatedCollection;
            }
            return;
        } catch (e) {
            return;
        }
    };

export const deleteCollectionTag = (key: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const item = getState().collectionPanel.item;
        const uuid = item ? item.uuid : '';
        try {
            if (item) {
                delete item.properties[key];
                const updatedCollection = await services.collectionService.update(uuid, item);
                dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Tag has been successfully deleted.", hideDuration: 2000 }));
                return updatedCollection;
            }
            return;
        } catch (e) {
            return;
        }
    };
