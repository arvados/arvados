// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from 'common/unionize';
import { extractUuidKind, Resource, ResourceWithProperties } from 'models/resource';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { getResourceService } from 'services/services';
import { addProperty, deleteProperty } from 'lib/resource-properties';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getResource } from './resources';
import { TagProperty } from 'models/tag';
import { change, formValueSelector } from 'redux-form';
import { ResourcePropertiesFormData } from 'views-components/resource-properties-form/resource-properties-form';

export const resourcesActions = unionize({
    SET_RESOURCES: ofType<Resource[]>(),
    DELETE_RESOURCES: ofType<string[]>()
});

export type ResourcesAction = UnionOf<typeof resourcesActions>;

export const updateResources = (resources: Resource[]) => resourcesActions.SET_RESOURCES(resources);

export const deleteResources = (resources: string[]) => resourcesActions.DELETE_RESOURCES(resources);

export const loadResource = (uuid: string, showErrors?: boolean) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const kind = extractUuidKind(uuid);
            const service = getResourceService(kind)(services);
            if (service) {
                const resource = await service.get(uuid, showErrors);
                dispatch<any>(updateResources([resource]));
                return resource;
            }
        } catch {}
        return undefined;
    };

export const deleteResourceProperty = (uuid: string, key: string, value: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();

        const rsc = getResource(uuid)(resources) as ResourceWithProperties;
        if (!rsc) { return; }

        const kind = extractUuidKind(uuid);
        const service = getResourceService(kind)(services);
        if (!service) { return; }

        const properties = Object.assign({}, rsc.properties);

        try {
            let updatedRsc = await service.update(
                uuid, {
                    properties: deleteProperty(properties, key, value),
                });
            updatedRsc = {...rsc, ...updatedRsc};
            dispatch<any>(updateResources([updatedRsc]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Property has been successfully deleted.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: e.errors[0], hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const createResourceProperty = (data: TagProperty) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { uuid } = data;
        const { resources } = getState();

        const rsc = getResource(uuid)(resources) as ResourceWithProperties;
        if (!rsc) { return; }

        const kind = extractUuidKind(uuid);
        const service = getResourceService(kind)(services);
        if (!service) { return; }

        try {
            const key = data.keyID || data.key;
            const value = data.valueID || data.value;
            const properties = Object.assign({}, rsc.properties);
            let updatedRsc = await service.update(
                rsc.uuid, {
                    properties: addProperty(properties, key, value),
                }
            );
            updatedRsc = {...rsc, ...updatedRsc};
            dispatch<any>(updateResources([updatedRsc]));
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Property has been successfully added.", hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            const errorMsg = e.errors && e.errors.length > 0 ? e.errors[0] : "Error while adding property";
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: errorMsg, hideDuration: 2000, kind: SnackbarKind.ERROR }));
        }
    };

export const addPropertyToResourceForm = (data: ResourcePropertiesFormData, formName: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const properties = { ...formValueSelector(formName)(getState(), 'properties') };
        const key = data.keyID || data.key;
        const value =  data.valueID || data.value;
        dispatch(change(
            formName,
            'properties',
            addProperty(properties, key, value)));
    };

export const removePropertyFromResourceForm = (key: string, value: string, formName: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const properties = { ...formValueSelector(formName)(getState(), 'properties') };
        dispatch(change(
            formName,
            'properties',
            deleteProperty(properties, key, value)));
    };
