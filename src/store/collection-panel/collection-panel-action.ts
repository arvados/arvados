// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { loadCollectionFiles } from "./collection-panel-files/collection-panel-files-actions";
import { CollectionResource } from '~/models/collection';
import { collectionPanelFilesAction } from "./collection-panel-files/collection-panel-files-actions";
import { createTree } from "~/models/tree";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { TagProperty } from "~/models/tag";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { resourcesActions } from "~/store/resources/resources-actions";
import { unionize, ofType, UnionOf } from '~/common/unionize';
import { SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { navigateTo } from '~/store/navigation/navigation-action';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';

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
        dispatch(loadDetailsPanel(collection.uuid));
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
                const updatedCollection = await services.collectionService.update(
                    uuid, {
                        properties: {
                            ...JSON.parse(JSON.stringify(item.properties)),
                            [data.key]: data.value
                        }
                    }
                );
                item.properties[data.key] = data.value;
                dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Tag has been successfully added.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
                return updatedCollection;
            }
            return;
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.errors[0], hideDuration: 2000, kind: SnackbarKind.ERROR }));
            return;
        }
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

export const deleteCollectionTag = (key: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const item = getState().collectionPanel.item;
        const uuid = item ? item.uuid : '';
        try {
            if (item) {
                delete item.properties[key];
                const updatedCollection = await services.collectionService.update(
                    uuid, {
                        properties: {...item.properties}
                    }
                );
                dispatch(resourcesActions.SET_RESOURCES([updatedCollection]));
                dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Tag has been successfully deleted.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
                return updatedCollection;
            }
            return;
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.errors[0], hideDuration: 2000, kind: SnackbarKind.ERROR }));
            return;
        }
    };
