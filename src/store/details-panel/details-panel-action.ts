// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { RootState } from '~/store/store';
import { Dispatch } from 'redux';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { getResource } from '~/store/resources/resources';
import { ProjectResource } from "~/models/project";
import { ServiceRepository } from '~/services/services';
import { TagProperty } from '~/models/tag';
import { startSubmit, stopSubmit } from 'redux-form';
import { resourcesActions } from '~/store/resources/resources-actions';
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';
import { addProperty, deleteProperty } from '~/lib/resource-properties';

export const SLIDE_TIMEOUT = 500;

export const detailsPanelActions = unionize({
    TOGGLE_DETAILS_PANEL: ofType<{}>(),
    OPEN_DETAILS_PANEL: ofType<string>(),
    LOAD_DETAILS_PANEL: ofType<string>()
});

export type DetailsPanelAction = UnionOf<typeof detailsPanelActions>;

export const PROJECT_PROPERTIES_FORM_NAME = 'projectPropertiesFormName';
export const PROJECT_PROPERTIES_DIALOG_NAME = 'projectPropertiesDialogName';

export const loadDetailsPanel = (uuid: string) => detailsPanelActions.LOAD_DETAILS_PANEL(uuid);

export const openDetailsPanel = (uuid: string) => detailsPanelActions.OPEN_DETAILS_PANEL(uuid);

export const openProjectPropertiesDialog = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(dialogActions.OPEN_DIALOG({ id: PROJECT_PROPERTIES_DIALOG_NAME, data: { } }));
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
export const toggleDetailsPanel = () => (dispatch: Dispatch) => {
    // because of material-ui issue resizing details panel breaks tabs.
    // triggering window resize event fixes that.
    setTimeout(() => {
        window.dispatchEvent(new Event('resize'));
    }, SLIDE_TIMEOUT);
    dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
};
