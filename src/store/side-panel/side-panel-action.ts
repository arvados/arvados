// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { isSidePanelTreeCategory, SidePanelTreeCategory } from '~/store/side-panel-tree/side-panel-tree-actions';
import { navigateToFavorites, navigateTo, navigateToTrash, navigateToSharedWithMe, navigateToWorkflows, navigateToPublicFavorites, navigateToAllProcesses } from '~/store/navigation/navigation-action';
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';

export const navigateFromSidePanel = (id: string) =>
    (dispatch: Dispatch) => {
        if (isSidePanelTreeCategory(id)) {
            dispatch<any>(getSidePanelTreeCategoryAction(id));
        } else {
            dispatch<any>(navigateTo(id));
        }
    };

const getSidePanelTreeCategoryAction = (id: string) => {
    switch (id) {
        case SidePanelTreeCategory.FAVORITES:
            return navigateToFavorites;
        case SidePanelTreeCategory.PUBLIC_FAVORITES:
            return navigateToPublicFavorites;
        case SidePanelTreeCategory.TRASH:
            return navigateToTrash;
        case SidePanelTreeCategory.SHARED_WITH_ME:
            return navigateToSharedWithMe;
        case SidePanelTreeCategory.WORKFLOWS:
            return navigateToWorkflows;
        case SidePanelTreeCategory.ALL_PROCESSES:
            return navigateToAllProcesses;
        default:
            return sidePanelTreeCategoryNotAvailable(id);
    }
};

const sidePanelTreeCategoryNotAvailable = (id: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `${id} not available`,
        hideDuration: 3000,
        kind: SnackbarKind.ERROR
    });
