// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from 'common/unionize';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { dialogActions } from 'store/dialog/dialog-actions';
import { getResource } from 'store/resources/resources';
import { ProjectResource } from "models/project";
import { ServiceRepository } from 'services/services';
import { TagProperty } from 'models/tag';
import { startSubmit, stopSubmit } from 'redux-form';
import { resourcesActions } from 'store/resources/resources-actions';
import {snackbarActions, SnackbarKind} from 'store/snackbar/snackbar-actions';
import { addProperty, deleteProperty } from 'lib/resource-properties';
import { FilterBuilder } from 'services/api/filter-builder';
import { OrderBuilder } from 'services/api/order-builder';
import { CollectionResource } from 'models/collection';
import { extractUuidKind, ResourceKind } from 'models/resource';

export const SLIDE_TIMEOUT = 500;

export const detailsPanelActions = unionize({
    TOGGLE_DETAILS_PANEL: ofType<{}>(),
    OPEN_DETAILS_PANEL: ofType<number>(),
    LOAD_DETAILS_PANEL: ofType<string>()
});

export type DetailsPanelAction = UnionOf<typeof detailsPanelActions>;

export const PROJECT_PROPERTIES_FORM_NAME = 'projectPropertiesFormName';
export const PROJECT_PROPERTIES_DIALOG_NAME = 'projectPropertiesDialogName';

export const loadDetailsPanel = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        if (getState().detailsPanel.isOpened) {
            switch(extractUuidKind(uuid)) {
                case ResourceKind.COLLECTION:
                    const c = getResource<CollectionResource>(uuid)(getState().resources);
                    dispatch<any>(refreshCollectionVersionsList(c!.currentVersionUuid));
                    break;
                default:
                    break;
            }
        }
        dispatch(detailsPanelActions.LOAD_DETAILS_PANEL(uuid));
    };

export const openDetailsPanel = (uuid?: string, tabNr: number = 0) =>
    (dispatch: Dispatch) => {
        dispatch(detailsPanelActions.OPEN_DETAILS_PANEL(tabNr));
        if (uuid !== undefined) {
            dispatch<any>(loadDetailsPanel(uuid));
        }
    };

export const openProjectPropertiesDialog = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(dialogActions.OPEN_DIALOG({ id: PROJECT_PROPERTIES_DIALOG_NAME, data: { } }));
    };

export const refreshCollectionVersionsList = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        services.collectionService.list({
            filters: new FilterBuilder()
                .addEqual('current_version_uuid', uuid)
                .getFilters(),
            includeOldVersions: true,
            order: new OrderBuilder<CollectionResource>().addDesc("version").getOrder()
        }).then(versions => dispatch(resourcesActions.SET_RESOURCES(versions.items))
        ).catch(e => snackbarActions.OPEN_SNACKBAR({
            message: `Couldn't retrieve versions: ${e.errors[0]}`,
            hideDuration: 2000,
            kind: SnackbarKind.ERROR })
        );
    };

export const deleteProjectProperty = (key: string, value: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { detailsPanel, resources } = getState();
        const project = getResource(detailsPanel.resourceUuid)(resources) as ProjectResource;
        if (!project) { return; }

        const properties = Object.assign({}, project.properties);

        try {
            const updatedProject = await services.projectService.update(
                project.uuid, {
                    properties: deleteProperty(properties, key, value),
                });
            dispatch(resourcesActions.SET_RESOURCES([updatedProject]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Property has been successfully deleted.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.errors[0], hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const createProjectProperty = (data: TagProperty) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { detailsPanel, resources } = getState();
        const project = getResource(detailsPanel.resourceUuid)(resources) as ProjectResource;
        if (!project) { return; }

        dispatch(startSubmit(PROJECT_PROPERTIES_FORM_NAME));
        try {
            const key = data.keyID || data.key;
            const value = data.valueID || data.value;
            const properties = Object.assign({}, project.properties);
            const updatedProject = await services.projectService.update(
                project.uuid, {
                    properties: addProperty(properties, key, value),
                }
            );
            dispatch(resourcesActions.SET_RESOURCES([updatedProject]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Property has been successfully added.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
            dispatch(stopSubmit(PROJECT_PROPERTIES_FORM_NAME));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.errors[0], hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };
export const toggleDetailsPanel = () => (dispatch: Dispatch, getState: () => RootState) => {
    // because of material-ui issue resizing details panel breaks tabs.
    // triggering window resize event fixes that.
    setTimeout(() => {
        window.dispatchEvent(new Event('resize'));
    }, SLIDE_TIMEOUT);
    dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
    if (getState().detailsPanel.isOpened) {
        dispatch<any>(loadDetailsPanel(getState().detailsPanel.resourceUuid));
    }
};
