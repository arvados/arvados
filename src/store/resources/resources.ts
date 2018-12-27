// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "~/models/resource";
import { ResourceKind } from '~/models/resource';

export type ResourcesState = { [key: string]: Resource };

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
