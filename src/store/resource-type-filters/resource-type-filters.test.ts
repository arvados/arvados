// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getInitialResourceTypeFilters, serializeResourceTypeFilters, ObjectTypeFilter, CollectionTypeFilter, ProcessTypeFilter, GroupTypeFilter } from './resource-type-filters';
import { ResourceKind } from 'models/resource';
import { deselectNode } from 'models/tree';
import { pipe } from 'lodash/fp';

describe("serializeResourceTypeFilters", () => {
    it("should serialize all filters", () => {
        const filters = getInitialResourceTypeFilters();
        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.PROCESS}","${ResourceKind.COLLECTION}"]]`);
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

    it("should serialize general and log collections", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROJECT),
            deselectNode(ObjectTypeFilter.PROCESS),
            deselectNode(CollectionTypeFilter.OUTPUT_COLLECTION)
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.COLLECTION}"]],["collections.properties.type","not in",["output"]]`);
    });

    it("should serialize only main processes", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROJECT),
            deselectNode(ProcessTypeFilter.CHILD_PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION)
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROCESS}"]],["container_requests.requesting_container_uuid","=",null]`);
    });

    it("should serialize only child processes", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROJECT),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION)
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROCESS}"]],["container_requests.requesting_container_uuid","!=",null]`);
    });

    it("should serialize all project types", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]]`);
    });

    it("should serialize filter groups", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(GroupTypeFilter.PROJECT)
            deselectNode(ObjectTypeFilter.PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]],["groups.group_class","=","filter"]`);
    });

    it("should serialize projects (normal)", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(GroupTypeFilter.FILTER_GROUP)
            deselectNode(ObjectTypeFilter.PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]],["groups.group_class","=","project"]`);
    });

});
