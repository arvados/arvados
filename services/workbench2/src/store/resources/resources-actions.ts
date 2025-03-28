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
import { showErrorSnackbar, showSuccessSnackbar, snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getResource } from './resources';
import { TagProperty } from 'models/tag';
import { change, formValueSelector } from 'redux-form';
import { ResourcePropertiesFormData } from 'views-components/resource-properties-form/resource-properties-form';
import { CommonResourceServiceError, getCommonResourceServiceError } from 'services/common-service/common-resource-service';

export type ResourceWithDescription = Resource & { description?: string }

export const resourcesActions = unionize({
    SET_RESOURCES: ofType<ResourceWithDescription[] >(),
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
        const vocabulary = getState().properties.vocabulary?.tags;
        const dataTags = getTagsIfExist(data.key, data.value, vocabulary);
        const key = data.keyID || dataTags.key || data.key;
        const value =  data.valueID || dataTags.value || data.value;
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


const getTagsIfExist = (dataKey: string, dataValue: string, vocabulary: any) => {
    let k, v;
    for (const key in vocabulary) {
        if (vocabulary[key].labels.find(l=>l.label === dataKey)) {
            k = key;
            const { values } = vocabulary[key];
            for (const val in values) {
                if (values[val].labels.find(l=>l.label === dataValue)) {
                    v = val;
                    break;
                }
            }
        }
    }
    return { key: k, value: v };
};

/**
 * Holds a map of CommonResourceServiceError types to error messages
 * Allows consumers to easily specify error messages to be grouped and displayed
 * for a batch resource operation
 */
export type CommonResourceErrorMessageFuncMap = {
    [key in CommonResourceServiceError]?: (count: number) => string;
}

/**
 * Utility type to hold a group of rejected promises associated with their error type
 */
type CommonResourceErrorResultMap = {
    [key in CommonResourceServiceError]?: PromiseRejectedResult[];
};

/**
 * Just a small type to tie the return generic of the success result with the passed in promise array
 */
export type SettledPromiseSet<T> = {
    success: PromiseFulfilledResult<T>[];
    error: PromiseRejectedResult[];
};

/**
 * Accepts a batched set of settled CommonResource Promise results and displays grouped error / success messages
 * @param promiseResults Array of allSettled promise results to be processed
 * @param messageFuncMap Map of CommonResourceServiceErrors to error message generator funcs
 * @param showSuccess Func called to show a success toast with message
 * @param showError Func called to show an error toast with message
 * @returns The separated success / error Promise results for further use
 */
export const showGroupedCommonResourceResultSnackbars = <T>(
    dispatch: Dispatch,
    promiseResults: PromiseSettledResult<T>[],
    messageFuncMap: CommonResourceErrorMessageFuncMap,
): SettledPromiseSet<T> => {
    // Split success and error promise results
    // Gets returned for the consumer to use (update stores, refresh DEs, etc)
    const success = promiseResults.filter((promiseResult): promiseResult is PromiseFulfilledResult<T> => promiseResult.status === 'fulfilled');
    const error = promiseResults.filter((promiseResult): promiseResult is PromiseRejectedResult => promiseResult.status === 'rejected');

    // Get the list of error types that we have error messages for
    const errorTypesWithMessages = (Object.keys(messageFuncMap) as Array<keyof typeof messageFuncMap>);

    // Group error promises by each CommonResourceError for which we have an associated message
    const mappedErrors = errorTypesWithMessages.map((key: CommonResourceServiceError): CommonResourceErrorResultMap => {
        // Filter the promises that match this error type
        const matchingPriomiseResults = error.filter((promiseResult) => {
            const errorType = getCommonResourceServiceError(promiseResult.reason);
            return (
                errorType === key &&
                // NONE is used for success, filter out any rejected promises that lack errors
                key !== CommonResourceServiceError.NONE &&
                // UNKNOWN is excluded and bundled with types that lack a message
                key !== CommonResourceServiceError.UNKNOWN
            );
        });
        return {[key]: matchingPriomiseResults};
    }).reduce((acc, curr) => {
        // Merge each error type -> promise result array into a single CommonResourceErrorResultMap object
        return Object.assign(acc, curr);
    }, {} as CommonResourceErrorResultMap);

    // Any errors not handled by the errorMessageMap are bundled into a generic error along with UNKNOWN
    const genericErrors = error.filter((promiseResult) => {
        return !Object.keys(messageFuncMap).includes(getCommonResourceServiceError(promiseResult.reason));
    });

    // Display success messages
    if (success.length) {
        const messageFunc = messageFuncMap[CommonResourceServiceError.NONE]
        if (messageFunc) {
            // Use NONE message func passed in for success message
            dispatch(showSuccessSnackbar(messageFunc(success.length)));
        } else {
            const itemText = success.length > 1 ? "items" : "item";
            dispatch(showSuccessSnackbar(`Operation successful (${success.length} ${itemText})`));
        }
    }

    const errorTypesFromErrors = Object.keys(mappedErrors) as Array<keyof typeof mappedErrors>;

    for(const errorType of errorTypesFromErrors) {
        const messageFunc = messageFuncMap[errorType];
        const errors = mappedErrors[errorType];

        // Errors here were included in the map so they should always have a messageFunc
        if (messageFunc && errors?.length) {
            dispatch(showErrorSnackbar(messageFunc(errors.length)));
        }
    }

    if (genericErrors.length) {
        const messageFunc = messageFuncMap[CommonResourceServiceError.UNKNOWN]
        if (messageFunc) {
            // Use UNKNOWN messageFunc for generic+unknown errors if provided
            dispatch(showErrorSnackbar(messageFunc(genericErrors.length)));
        } else {
            const itemText = genericErrors.length > 1 ? "items" : "item";
            dispatch(showErrorSnackbar(`Operation failed (${genericErrors.length} ${itemText})`));
        }
    }

    return { success, error };
};
