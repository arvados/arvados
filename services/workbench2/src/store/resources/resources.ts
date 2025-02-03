// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, EditableResource } from "models/resource";
import { ResourceKind } from 'models/resource';
import { GroupResource } from "models/group";
import { memoize } from "lodash";

export type ResourcesState = { [key: string]: Resource };

export const getResourceWithEditableStatus = <T extends GroupResource & EditableResource>(id: string, userUuid?: string) =>
    (state: ResourcesState): T | undefined => {
        if (state[id] === undefined) { return; }

        const resource = JSON.parse(JSON.stringify(state[id])) as T;

        if (resource) {
            if (resource.canWrite === undefined) {
                resource.isEditable = (state[resource.ownerUuid] as GroupResource)?.canWrite;
            } else {
                resource.isEditable = resource.canWrite;
            }
        }

        return resource;
    };

export const getResource = memoize(<T extends Resource = Resource>(id: string) =>
    memoize((state: ResourcesState): T | undefined =>
        state[id] as T)
);

export const getResourceFromState = memoize(<T extends Resource = Resource>(id: string) =>
    memoize((resources: ResourcesState) =>
    resources[id] as T | undefined)
);

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
    (state: ResourcesState) => {
        const items: Resource[] = [];
        for (const id in state) {
            const resource = state[id];
            if (resource && filter(resource)) {
                items.push(resource);
            }
        }
        return items;
    };

export const filterResourcesByKind = (kind: ResourceKind) =>
    (state: ResourcesState) =>
        filterResources(resource => resource.kind === kind)(state);
