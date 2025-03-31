// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { trashPanelActions } from "store/trash-panel/trash-panel-action";
import { activateSidePanelTreeItem, loadSidePanelTreeProjects, SidePanelTreeCategory } from "store/side-panel-tree/side-panel-tree-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { sharedWithMePanelActions } from "store/shared-with-me-panel/shared-with-me-panel-actions";
import { extractUuidKind, ResourceKind } from "models/resource";
import { navigateTo } from "store/navigation/navigation-action";
import { matchFavoritesRoute, matchProjectRoute, matchSharedWithMeRoute, matchTrashRoute } from "routes/routes";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { addDisabledButton } from "store/multiselect/multiselect-actions";
import { showGroupedCommonResourceResultSnackbars, updateResources } from "store/resources/resources-actions";
import { favoritePanelActions } from "store/favorite-panel/favorite-panel-action";
import { CommonResourceServiceError } from "services/common-service/common-resource-service";
import _ from "lodash";

/**
 * Toggles the trash status of an array of UUIDS based on the current isTrashed status
 * @param uuids list of uuids to trash/untrash
 * @param isTrashed Current trashed status to be toggled
 * @returns Dispatchable action that yields a void promise
 *
 * This only handles trashable resources aka Collection / Group
 */
export const toggleResourceTrashed =
    (uuids: string[], isTrashed: boolean) =>
        async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
            dispatch<any>(addDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH));

            const verb = isTrashed ? "untrash" : "trash";
            const messageFuncMap = {
                [CommonResourceServiceError.NONE]: (count: number) => count > 1 ? `${_.startCase(verb)}ed ${count} items` : `Item ${verb}ed`,
                [CommonResourceServiceError.PERMISSION_ERROR_FORBIDDEN]: (count: number) => count > 1 ? `${_.startCase(verb)} ${count} items failed: Access Denied` : `${_.startCase(verb)} failed: Access Denied`,
                [CommonResourceServiceError.UNIQUE_NAME_VIOLATION]: (count: number) => count > 1 ? `${_.startCase(verb)} ${count} items failed: Duplicate Name` : `${_.startCase(verb)} failed: Duplicate Name`,
                [CommonResourceServiceError.UNKNOWN]: (count: number) => count > 1 ? `${_.startCase(verb)} ${count} items failed` : `${_.startCase(verb)} failed`,
            };

            const trashFunc = async (uuid: string) => {
                const kind = extractUuidKind(uuid);
                if (kind === ResourceKind.COLLECTION) {
                    return isTrashed ? services.collectionService.untrash(uuid) : services.collectionService.trash(uuid);
                } else if (kind === ResourceKind.GROUP) {
                    return isTrashed ? services.groupsService.untrash(uuid) : services.groupsService.trash(uuid);
                }
                console.error("Trash operation failed: resource type not trashable " + uuid);
                return Promise.reject();
            };

            await Promise.allSettled(uuids.map((uuid) => trashFunc(uuid)))
                .then(async settledPromises => {
                    const { success } = showGroupedCommonResourceResultSnackbars(dispatch, settledPromises, messageFuncMap);

                    if (success.length) {
                        const { location } = getState().router;
                        // Update store
                        await dispatch<any>(updateResources(success.map(success => success.value)));
                        if (isTrashed) {
                            // Refresh trash panel after untrash
                            if (matchTrashRoute(location ? location.pathname : "")) {
                                dispatch(trashPanelActions.REQUEST_ITEMS());
                            }
                            // Navigate to untrashed project when only item
                            if (uuids.length === 1 && success.length === 1) {
                                const uuid = success[0].value.uuid;
                                if (extractUuidKind(uuid) === ResourceKind.GROUP) {
                                    dispatch<any>(navigateTo(uuid));
                                    dispatch<any>(activateSidePanelTreeItem(uuid));
                                }
                            }
                            // Reload favorites
                            dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES));
                        } else {
                            // Refresh favorites / project view after trashed
                            if (matchFavoritesRoute(location ? location.pathname : "")) {
                                dispatch(favoritePanelActions.REQUEST_ITEMS());
                            } else if (matchProjectRoute(location ? location.pathname : "")) {
                                dispatch(projectPanelDataActions.REQUEST_ITEMS());
                            } else if (matchSharedWithMeRoute(location ? location.pathname : "")) {
                                dispatch(sharedWithMePanelActions.REQUEST_ITEMS());
                            }

                            // If 1 item trashed, navigate to parent
                            if (uuids.length === 1 && success.length === 1) {
                                dispatch<any>(navigateTo(success[0].value.ownerUuid));
                            }

                            // Reload favorites
                            dispatch<any>(loadSidePanelTreeProjects(SidePanelTreeCategory.FAVORITES))
                                // Using then to ensure loadSidePanelTreeProjects finished
                                .then(() => {
                                    // Refresh each project's parent in the side panel tree
                                    // Get every successfully untrashed reasource
                                    success.map(result => result.value)
                                        // Filter to only GROUP (project)
                                        .filter(resource => resource.kind === ResourceKind.GROUP)
                                        // Load side panel for each
                                        .map(resource =>
                                            dispatch<any>(loadSidePanelTreeProjects(resource.ownerUuid))
                                        );
                                });
                        }
                    }
                });
        };
