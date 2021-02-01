// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import {
    loadCollectionFiles,
    COLLECTION_PANEL_LOAD_FILES_THRESHOLD
} from "./collection-panel-files/collection-panel-files-actions";
import { CollectionResource } from '~/models/collection';
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { TagProperty } from "~/models/tag";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { resourcesActions } from "~/store/resources/resources-actions";
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { navigateTo } from '~/store/navigation/navigation-action';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { addProperty, deleteProperty } from "~/lib/resource-properties";

export const collectionPanelActions = unionize({
    SET_COLLECTION: ofType<CollectionResource>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
    LOAD_BIG_COLLECTIONS: ofType<boolean>(),
});

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const COLLECTION_TAG_FORM_NAME = 'collectionTagForm';

export const loadCollectionPanel = (uuid: string, forceReload = false) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { collectionPanel: { item } } = getState();
        const collection = (item && item.uuid === uuid && !forceReload)
            ? item
            : await services.collectionService.get(uuid);
        dispatch<any>(loadDetailsPanel(collection.uuid));
        dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item: collection }));
        dispatch(resourcesActions.SET_RESOURCES([collection]));
        if (collection.fileCount <= COLLECTION_PANEL_LOAD_FILES_THRESHOLD &&
            !getState().collectionPanel.loadBigCollections) {
            dispatch<any>(loadCollectionFiles(collection.uuid));
        }
        return collection;
    };

export const createCollectionTag = (data: TagProperty) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const item = getState().collectionPanel.item;
        if (!item) { return; }

        const properties = Object.assign({}, item.properties);
        const key = data.keyID || data.key;
        const value = data.valueID || data.value;
        services.collectionService.update(
            item.uuid, {
                properties: addProperty(properties, key, value)
            }
        ).then(updatedCollection => {
            dispatch(collectionPanelActions.SET_COLLECTION(updatedCollection));
            dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Property has been successfully added.",
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS }));
            dispatch<any>(loadDetailsPanel(updatedCollection.uuid));
            return updatedCollection;
        }).catch (e =>
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: e.errors[0],
                hideDuration: 2000,
                kind: SnackbarKind.ERROR }))
        );
    };

export const navigateToProcess = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        try {
            await services.containerRequestService.get(uuid);
            dispatch<any>(navigateTo(uuid));
        } catch {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'This process does not exist!', hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const deleteCollectionTag = (key: string, value: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const item = getState().collectionPanel.item;
        if (!item) { return; }

        const properties = Object.assign({}, item.properties);
        services.collectionService.update(
            item.uuid, {
                properties: deleteProperty(properties, key, value)
            }
        ).then(updatedCollection => {
            dispatch(collectionPanelActions.SET_COLLECTION(updatedCollection));
            dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Tag has been successfully deleted.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch<any>(loadDetailsPanel(updatedCollection.uuid));
            return updatedCollection;
        }).catch (e => {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: e.errors[0],
                hideDuration: 2000,
                kind: SnackbarKind.ERROR }));
        });
    };
