// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, EditableResource } from "models/resource";
import { ResourceKind } from 'models/resource';
import { ProjectResource } from "models/project";
import { GroupResource } from "models/group";

export type ResourcesState = { [key: string]: Resource };

const getResourceWritableBy = (state: ResourcesState, id: string, userUuid: string): string[] => {
    if (!id) {
        return [];
    }

    if (id === userUuid) {
        return [userUuid];
    }

    const resource = (state[id] as ProjectResource);

    if (!resource) {
        return [];
    }

    const { writableBy } = resource;

    return writableBy || getResourceWritableBy(state, resource.ownerUuid, userUuid);
};

export const getResourceWithEditableStatus = <T extends EditableResource & GroupResource>(id: string, userUuid?: string) =>
    (state: ResourcesState): T | undefined => {
        if (state[id] === undefined) { return; }

        const resource = JSON.parse(JSON.stringify(state[id] as T));

        if (resource) {
            resource.isEditable = userUuid ? getResourceWritableBy(state, id, userUuid).indexOf(userUuid) > -1 : false;
        }

        return resource;
    };

export const getResource = <T extends Resource = Resource>(id: string) =>
    (state: ResourcesState): T | undefined =>
        state[id] as T;

export const setResource = <T extends Resource>(id: string, data: T) =>
    (state: ResourcesState) => ({
        ...state,
        [id]: data
    });

export const deleteResource = (id: string) =>
    (state: ResourcesState) => {
        const newState = { ...state };
        delete newState[id];
        return newState;
    };

export const filterResources = (filter: (resource: Resource) => boolean) =>
    (state: ResourcesState) =>
        Object
            .keys(state)
            .reduce((resources, id) => {
                const resource = getResource(id)(state);
                return resource
                    ? [...resources, resource]
                    : resources;
            }, [])
            .filter(filter);

export const filterResourcesByKind = (kind: ResourceKind) =>
    (state: ResourcesState) =>
        filterResources(resource => resource.kind === kind)(state);
