// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getInitialResourceTypeFilters, serializeResourceTypeFilters, ObjectTypeFilter, CollectionTypeFilter } from './resource-type-filters';
import { ResourceKind } from '~/models/resource';
import { deselectNode } from '~/models/tree';
import { pipe } from 'lodash/fp';

describe("serializeResourceTypeFilters", () => {
    it("should serialize all filters", () => {
        const filters = getInitialResourceTypeFilters();
        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.PROCESS}","${ResourceKind.COLLECTION}"]],["collections.properties.type","in",["nil","output","log"]]`);
    });

    it("should serialize all but collection filters", () => {
        const filters = deselectNode(ObjectTypeFilter.COLLECTION)(getInitialResourceTypeFilters());
        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.PROCESS}"]]`);
    });

    it("should serialize output collections and projects", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROCESS),
            deselectNode(CollectionTypeFilter.GENERAL_COLLECTION),
            deselectNode(CollectionTypeFilter.LOG_COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.COLLECTION}"]],["collections.properties.type","in",["output"]]`);
    });
});
