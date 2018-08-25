// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { isSidePanelTreeCategory, SidePanelTreeCategory } from '~/store/side-panel-tree/side-panel-tree-actions';
import { navigateToFavorites, navigateTo } from '../navigation/navigation-action';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { RootState } from '~/store/store';
import { extractUuidKind, ResourceKind } from '~/models/resource';
import { openProjectContextMenu } from '~/store/context-menu/context-menu-actions';
import { openRootProjectContextMenu } from '../context-menu/context-menu-actions';

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
        default:
            return sidePanelTreeCategoryNotAvailable(id);
    }
};

const sidePanelTreeCategoryNotAvailable = (id: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `${id} not available`,
        hideDuration: 3000,
    });