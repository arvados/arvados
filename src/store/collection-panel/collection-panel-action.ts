// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import {
    COLLECTION_PANEL_LOAD_FILES_THRESHOLD
} from "./collection-panel-files/collection-panel-files-actions";
import { CollectionResource } from 'models/collection';
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { resourcesActions } from "store/resources/resources-actions";
import { unionize, ofType, UnionOf } from 'common/unionize';
import { SnackbarKind } from 'store/snackbar/snackbar-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';

export const collectionPanelActions = unionize({
    SET_COLLECTION: ofType<CollectionResource>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
    LOAD_BIG_COLLECTIONS: ofType<boolean>(),
});

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

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
        }
        return collection;
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
