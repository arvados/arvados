// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { push } from "react-router-redux";
import { RootState } from "../store";
import { ResourceKind, Resource, extractUuidKind } from '~/models/resource';
import { getCollectionUrl } from "~/models/collection";
import { getProjectUrl } from "~/models/project";
import { getResource } from '~/store/resources/resources';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { loadCollectionPanel } from '~/store/collection-panel/collection-panel-action';
import { snackbarActions } from '../snackbar/snackbar-actions';
import { resourceLabel } from "~/common/labels";
import { loadFavoritePanel } from '../favorite-panel/favorite-panel-action';
import { openProjectPanel, projectPanelActions } from '~/store/project-panel/project-panel-action';
import { activateSidePanelTreeItem, initSidePanelTree, SidePanelTreeCategory } from '../side-panel-tree/side-panel-tree-actions';
import { Routes } from '~/routes/routes';
import { loadResource } from '../resources/resources-actions';
import { ServiceRepository } from '~/services/services';
import { favoritePanelActions } from '~/store/favorite-panel/favorite-panel-action';
import { projectPanelColumns } from '~/views/project-panel/project-panel';
import { favoritePanelColumns } from '~/views/favorite-panel/favorite-panel';
import { matchRootRoute } from '~/routes/routes';
import { setCollectionBreadcrumbs, setProjectBreadcrumbs, setSidePanelBreadcrumbs } from '../breadcrumbs/breadcrumbs-actions';

export const navigateTo = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState) => {
        const kind = extractUuidKind(uuid);
        if (kind === ResourceKind.PROJECT || kind === ResourceKind.USER) {
            dispatch<any>(navigateToProject(uuid));
        } else if (kind === ResourceKind.COLLECTION) {
            dispatch<any>(navigateToCollection(uuid));
        }
        if (uuid === SidePanelTreeCategory.FAVORITES) {
            dispatch<any>(navigateToFavorites);
        }
    };

const getResourceNavigationAction = (resource: Resource) => {
    switch (resource.kind) {
        case ResourceKind.COLLECTION:
            return navigateToCollection(resource.uuid);
        case ResourceKind.PROJECT:
        case ResourceKind.USER:
            return navigateToProject(resource.uuid);
        default:
            return cannotNavigateToResource(resource);
    }
};

export const loadWorkbench = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { auth, router } = getState();
        const { user } = auth;
        if (user) {
            const userResource = await dispatch<any>(loadResource(user.uuid));
            if (userResource) {
                dispatch(projectPanelActions.SET_COLUMNS({ columns: projectPanelColumns }));
                dispatch(favoritePanelActions.SET_COLUMNS({ columns: favoritePanelColumns }));
                dispatch<any>(initSidePanelTree());
                if (router.location) {
                    const match = matchRootRoute(router.location.pathname);
                    if (match) {
                        dispatch(navigateToProject(userResource.uuid));
                    }
                }
            } else {
                dispatch(userIsNotAuthenticated);
            }
        } else {
            dispatch(userIsNotAuthenticated);
        }
    };

export const navigateToFavorites = push(Routes.FAVORITES);

export const loadFavorites = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch<any>(activateSidePanelTreeItem(SidePanelTreeCategory.FAVORITES));
        dispatch<any>(loadFavoritePanel());
        dispatch<any>(setSidePanelBreadcrumbs(SidePanelTreeCategory.FAVORITES));
    };


export const navigateToProject = compose(push, getProjectUrl);

export const loadProject = (uuid: string) =>
    async (dispatch: Dispatch) => {
        await dispatch<any>(activateSidePanelTreeItem(uuid));
        dispatch<any>(setProjectBreadcrumbs(uuid));
        dispatch<any>(openProjectPanel(uuid));
        dispatch(loadDetailsPanel(uuid));
    };

export const navigateToCollection = compose(push, getCollectionUrl);

export const loadCollection = (uuid: string) =>
    async (dispatch: Dispatch) => {
        const collection = await dispatch<any>(loadCollectionPanel(uuid));
        await dispatch<any>(activateSidePanelTreeItem(collection.ownerUuid));
        dispatch<any>(setCollectionBreadcrumbs(collection.uuid));
        dispatch(loadDetailsPanel(uuid));
    };

export const cannotNavigateToResource = ({ kind, uuid }: Resource) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `${resourceLabel(kind)} identified by ${uuid} cannot be opened.`
    });

export const resourceIsNotLoaded = (uuid: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Resource identified by ${uuid} is not loaded.`
    });

export const userIsNotAuthenticated = snackbarActions.OPEN_SNACKBAR({
    message: 'User is not authenticated'
});

export const couldNotLoadUser = snackbarActions.OPEN_SNACKBAR({
    message: 'Could not load user'
});