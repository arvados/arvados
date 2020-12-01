// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { snackbarActions, SnackbarKind } from "../snackbar/snackbar-actions";
import { resourcesActions } from "../resources/resources-actions";
import { navigateTo } from "../navigation/navigation-action";
import { dialogActions } from "../dialog/dialog-actions";

export const COLLECTION_RECOVER_VERSION_DIALOG = 'collectionRecoverVersionDialog';

export const openRecoverCollectionVersionDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: COLLECTION_RECOVER_VERSION_DIALOG,
            data: {
                title: 'Recover version',
                text: "Do you want to make this the new collection's head version? If you don't want to modify the current head version, you can just make a copy.",
                confirmButtonLabel: 'Recover',
                uuid
            }
        }));
    };

export const recoverVersion = (resourceUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            // Request que entire record because stored old versions usually
            // don't include the manifest_text field.
            const oldVersion = await services.collectionService.get(resourceUuid);
            const { uuid, version, ...rest} = oldVersion;
            const headVersion = await services.collectionService.update(
                oldVersion.currentVersionUuid,
                { ...rest }
            );
            dispatch(resourcesActions.SET_RESOURCES([headVersion]));
            dispatch<any>(navigateTo(headVersion.uuid));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `Couldn't recover version: ${e.errors[0]}`,
                hideDuration: 2000,
                kind: SnackbarKind.ERROR
            }));
        }
    };
