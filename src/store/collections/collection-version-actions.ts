// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import { resourcesActions } from "../resources/resources-actions";
import { navigateTo } from "../navigation/navigation-action";
import { dialogActions } from "../dialog/dialog-actions";
import { getResource } from "store/resources/resources";
import { CollectionResource } from "models/collection";

export const COLLECTION_RESTORE_VERSION_DIALOG = 'collectionRestoreVersionDialog';

export const openRestoreCollectionVersionDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: COLLECTION_RESTORE_VERSION_DIALOG,
            data: {
                title: 'Restore version',
                text: "This will copy the content of the selected version to the head. To make a new collection with the content of the selected version, use 'Make a copy' instead.",
                confirmButtonLabel: 'Restore',
                uuid
            }
        }));
    };

export const restoreVersion = (resourceUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            // Request the manifest text because stored old versions usually
            // don't include them.
            let oldVersion = getResource<CollectionResource>(resourceUuid)(getState().resources);
            if (!oldVersion) {
                oldVersion = await services.collectionService.get(resourceUuid);
            }
            const oldVersionManifest = await services.collectionService.get(resourceUuid, undefined, ['manifestText']);
            oldVersion.manifestText = oldVersionManifest.manifestText;

            const { uuid, version, ...rest} = oldVersion;
            const headVersion = await services.collectionService.update(
                oldVersion.currentVersionUuid,
                { ...rest }
            );
            dispatch(resourcesActions.SET_RESOURCES([headVersion]));
            dispatch<any>(navigateTo(headVersion.uuid));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Couldn't restore version: ${e.errors[0]}`,
                hideDuration: 2000,
                kind: SnackbarKind.ERROR
            }));
        }
    };
