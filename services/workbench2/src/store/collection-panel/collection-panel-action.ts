// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { CollectionResource } from 'models/collection';
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions } from "../snackbar/snackbar-actions";
import { resourcesActions } from "store/resources/resources-actions";
import { unionize, ofType, UnionOf } from 'common/unionize';
import { SnackbarKind } from 'store/snackbar/snackbar-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export const collectionPanelActions = unionize({
    SET_COLLECTION: ofType<CollectionResource>(),
});

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const loadCollectionPanel = (uuid: string, forceReload = false) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { collectionPanel: { item } } = getState();
        let collection: CollectionResource | null = null;
        if (!item || item.uuid !== uuid || forceReload) {
            try {
                dispatch(progressIndicatorActions.START_WORKING(uuid + "-panel"));
                collection = await services.collectionService.get(uuid);
                dispatch(collectionPanelActions.SET_COLLECTION(collection));
                dispatch(resourcesActions.SET_RESOURCES([collection]));
            } finally {
                dispatch(progressIndicatorActions.STOP_WORKING(uuid + "-panel"));
            }
        } else {
            collection = item;
        }
        dispatch<any>(loadDetailsPanel(collection.uuid));
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
