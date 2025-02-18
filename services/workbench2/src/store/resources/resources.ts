// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "models/resource";
import { ResourceKind } from 'models/resource';
import { memoize } from "lodash";

export type ResourcesState = { [key: string]: Resource };

export const getResource = memoize(<T extends Resource = Resource>(id: string | undefined) =>
    (state: ResourcesState): T | undefined =>
        id ? state[id] as T : undefined);

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
