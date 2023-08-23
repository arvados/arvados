// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getInitialResourceTypeFilters, serializeResourceTypeFilters, ObjectTypeFilter, CollectionTypeFilter, ProcessTypeFilter, GroupTypeFilter, buildProcessStatusFilters, ProcessStatusFilter } from './resource-type-filters';
import { ResourceKind } from 'models/resource';
import { selectNode, deselectNode } from 'models/tree';
import { pipe } from 'lodash/fp';
import { FilterBuilder } from 'services/api/filter-builder';

describe("buildProcessStatusFilters", () => {
    [
        [ProcessStatusFilter.ALL, ""],
        [ProcessStatusFilter.ONHOLD, `["state","!=","Final"],["priority","=","0"],["container.state","in",["Queued","Locked"]]`],
        [ProcessStatusFilter.COMPLETED, `["container.state","=","Complete"],["container.exit_code","=","0"]`],
        [ProcessStatusFilter.FAILED, `["container.state","=","Complete"],["container.exit_code","!=","0"]`],
        [ProcessStatusFilter.QUEUED, `["container.state","=","Queued"],["priority","!=","0"]`],
        [ProcessStatusFilter.CANCELLED, `["container.state","=","Cancelled"]`],
        [ProcessStatusFilter.RUNNING, `["container.state","=","Running"]`],
    ].forEach(([status, expected]) => {
        it(`can filter "${status}" processes`, () => {
            const filters = buildProcessStatusFilters(new FilterBuilder(), status);
            expect(filters.getFilters())
                .toEqual(expected);
        })
    });
});

describe("serializeResourceTypeFilters", () => {
    it("should serialize all filters", () => {
        const filters = getInitialResourceTypeFilters();
        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.COLLECTION}","${ResourceKind.WORKFLOW}","${ResourceKind.PROCESS}"]],["collections.properties.type","not in",["log","intermediate"]],["container_requests.requesting_container_uuid","=",null]`);
    });

    it("should serialize all but collection filters", () => {
        const filters = deselectNode(ObjectTypeFilter.COLLECTION)(getInitialResourceTypeFilters());
        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.WORKFLOW}","${ResourceKind.PROCESS}"]],["container_requests.requesting_container_uuid","=",null]`);
    });

    it("should serialize output collections and projects", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(CollectionTypeFilter.GENERAL_COLLECTION),
            deselectNode(CollectionTypeFilter.LOG_COLLECTION),
            deselectNode(CollectionTypeFilter.INTERMEDIATE_COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.COLLECTION}"]],["collections.properties.type","in",["output"]]`);
    });

    it("should serialize output collections and projects", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(CollectionTypeFilter.GENERAL_COLLECTION),
            deselectNode(CollectionTypeFilter.LOG_COLLECTION),
            deselectNode(CollectionTypeFilter.INTERMEDIATE_COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROJECT}","${ResourceKind.COLLECTION}"]],["collections.properties.type","in",["output"]]`);
    });

    it("should serialize general collections", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROJECT),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(CollectionTypeFilter.OUTPUT_COLLECTION)
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.COLLECTION}"]],["collections.properties.type","not in",["output","log","intermediate"]]`);
    });

    it("should serialize only main processes", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.PROJECT),
            deselectNode(ProcessTypeFilter.CHILD_PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
            deselectNode(ObjectTypeFilter.DEFINITION),
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
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ObjectTypeFilter.COLLECTION),

            selectNode(ProcessTypeFilter.CHILD_PROCESS),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.PROCESS}"]],["container_requests.requesting_container_uuid","!=",null]`);
    });

    it("should serialize all project types", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(ObjectTypeFilter.COLLECTION),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]]`);
    });

    it("should serialize filter groups", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(GroupTypeFilter.PROJECT),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]],["groups.group_class","=","filter"]`);
    });

    it("should serialize projects (normal)", () => {
        const filters = pipe(
            () => getInitialResourceTypeFilters(),
            deselectNode(GroupTypeFilter.FILTER_GROUP),
            deselectNode(ObjectTypeFilter.DEFINITION),
            deselectNode(ProcessTypeFilter.MAIN_PROCESS),
            deselectNode(ObjectTypeFilter.COLLECTION),
        )();

        const serializedFilters = serializeResourceTypeFilters(filters);
        expect(serializedFilters)
            .toEqual(`["uuid","is_a",["${ResourceKind.GROUP}"]],["groups.group_class","=","project"]`);
    });

});
