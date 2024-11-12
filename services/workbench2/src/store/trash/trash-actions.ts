// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { trashPanelActions } from "store/trash-panel/trash-panel-action";
import { activateSidePanelTreeItem, loadSidePanelTreeProjects, SidePanelTreeCategory } from "store/side-panel-tree/side-panel-tree-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { sharedWithMePanelActions } from "store/shared-with-me-panel/shared-with-me-panel-actions";
import { ResourceKind } from "models/resource";
import { navigateTo, navigateToTrash } from "store/navigation/navigation-action";
import { matchCollectionRoute, matchFavoritesRoute, matchProjectRoute, matchSharedWithMeRoute } from "routes/routes";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { addDisabledButton } from "store/multiselect/multiselect-actions";
import { updateResources } from "store/resources/resources-actions";
import { GroupResource } from "models/group";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { CollectionResource } from "models/collection";

export const toggleProjectTrashed =
    (uuid: string, ownerUuid: string, isTrashed: boolean, isMulti: boolean) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
            let errorMessage = "";
            let successMessage = "";
            let toggledResource: GroupResource | undefined = undefined;
            dispatch<any>(addDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH))
            try {
                if (isTrashed) {
                    errorMessage = "Could not restore project from trash";
                    successMessage = "Restored project from trash";
                    toggledResource = await services.groupsService.untrash(uuid);
                    if (toggledResource) {
                        // Resource store must be updated with trashed flag for favorites tree to hide trashed
                        await dispatch<any>(updateResources([toggledResource]));
                    }
                    dispatch<any>(isMulti || !toggledResource ? navigateToTrash : navigateTo(uuid));
                    dispatch<any>(activateSidePanelTreeItem(uuid));
                    dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES));
                } else {
                    errorMessage = "Could not move project to trash";
                    successMessage = "Added project to trash";
                    toggledResource = await services.groupsService.trash(uuid);
                    if (toggledResource) {
                        // Resource store must be updated with trashed flag for favorites tree to hide trashed
                        await dispatch<any>(updateResources([toggledResource]));
                    }
                    // Refresh favorites tree after trash/untrash
                    await dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES));
                    dispatch<any>(loadSidePanelTreeProjects(ownerUuid));

                    const { location } = getState().router;
                    if (matchSharedWithMeRoute(location ? location.pathname : "")) {
                        dispatch(sharedWithMePanelActions.REQUEST_ITEMS());
                    }
                    else {
                        dispatch<any>(navigateTo(ownerUuid));
                    }
                }
                if (toggledResource) {
                        dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: successMessage,
                            hideDuration: 2000,
                            kind: SnackbarKind.SUCCESS,
                        })
                    );
                }
            } catch (e) {
                if (e.status === 422) {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: "Could not restore project from trash: Duplicate name at destination",
                            kind: SnackbarKind.ERROR,
                        })
                    );
                } else {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: errorMessage,
                            kind: SnackbarKind.ERROR,
                        })
                    );
                }
            }
        };

export const toggleCollectionTrashed =
    (uuid: string, isTrashed: boolean) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
            let errorMessage = "";
            let successMessage = "";
            let toggledResource: CollectionResource | undefined = undefined;
            dispatch<any>(addDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH))
            try {
                if (isTrashed) {
                    const { location } = getState().router;
                    errorMessage = "Could not restore collection from trash";
                    successMessage = "Restored from trash";
                    toggledResource = await services.collectionService.untrash(uuid);
                    if (toggledResource) {
                        await dispatch<any>(updateResources([toggledResource]));
                    }
                    if (matchCollectionRoute(location ? location.pathname : "")) {
                        dispatch(navigateToTrash);
                    }
                    dispatch(trashPanelActions.REQUEST_ITEMS());
                    dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES));
                } else {
                    errorMessage = "Could not move collection to trash";
                    successMessage = "Added to trash";
                    toggledResource = await services.collectionService.trash(uuid);
                    if (toggledResource) {
                        await dispatch<any>(updateResources([toggledResource]));
                    }

                    const { location } = getState().router;
                    if (matchFavoritesRoute(location ? location.pathname : "")) {
                        dispatch(favoritePanelActions.REQUEST_ITEMS());
                    } else if (matchProjectRoute(location ? location.pathname : "")) {
                        dispatch(projectPanelDataActions.REQUEST_ITEMS());
                    }
                    dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES));
                }
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: successMessage,
                        hideDuration: 2000,
                        kind: SnackbarKind.SUCCESS,
                    })
                );
            } catch (e) {
                if (e.status === 422) {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: "Could not restore collection from trash: Duplicate name at destination",
                            kind: SnackbarKind.ERROR,
                        })
                    );
                } else {
                    dispatch(
                        snackbarActions.OPEN_SNACKBAR({
                            message: errorMessage,
                            kind: SnackbarKind.ERROR,
                        })
                    );
                }
            }
        };

export const toggleTrashed = (kind: ResourceKind, uuid: string, ownerUuid: string, isTrashed: boolean) => (dispatch: Dispatch) => {
    if (kind === ResourceKind.PROJECT) {
        dispatch<any>(toggleProjectTrashed(uuid, ownerUuid, isTrashed!!, false));
    } else if (kind === ResourceKind.COLLECTION) {
        dispatch<any>(toggleCollectionTrashed(uuid, isTrashed!!));
    }
};
