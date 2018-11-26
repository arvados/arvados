// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { FilterBuilder } from "./filter-builder";

describe("FilterBuilder", () => {

    let filters: FilterBuilder;

    beforeEach(() => {
        filters = new FilterBuilder();
    });

    it("should add 'equal' rule (string)", () => {
        expect(
            filters.addEqual("etag", "etagValue").getFilters()
        ).toEqual(`["etag","=","etagValue"]`);
    });

    it("should add 'equal' rule (boolean)", () => {
        expect(
            filters.addEqual("is_trashed", true).getFilters()
        ).toEqual(`["is_trashed","=",true]`);
    });

    it("should add 'like' rule", () => {
        expect(
            filters.addLike("etag", "etagValue").getFilters()
        ).toEqual(`["etag","like","%etagValue%"]`);
    });

    it("should add 'ilike' rule", () => {
        expect(
            filters.addILike("etag", "etagValue").getFilters()
        ).toEqual(`["etag","ilike","%etagValue%"]`);
    });

    it("should add 'is_a' rule", () => {
        expect(
            filters.addIsA("etag", "etagValue").getFilters()
        ).toEqual(`["etag","is_a","etagValue"]`);
    });

    it("should add 'is_a' rule for set", () => {
        expect(
            filters.addIsA("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).toEqual(`["etag","is_a",["etagValue1","etagValue2"]]`);
    });

    it("should add 'in' rule", () => {
        expect(
            filters.addIn("etag", "etagValue").getFilters()
        ).toEqual(`["etag","in","etagValue"]`);
    });

    it("should add 'in' rule for set", () => {
        expect(
            filters.addIn("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).toEqual(`["etag","in",["etagValue1","etagValue2"]]`);
    });

    it("should add 'not in' rule for set", () => {
        expect(
            filters.addNotIn("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).toEqual(`["etag","not in",["etagValue1","etagValue2"]]`);
    });

    it("should add multiple rules", () => {
        expect(
            filters
                .addIn("etag", ["etagValue1", "etagValue2"])
                .addEqual("href", "hrefValue")
                .getFilters()
        ).toEqual(`["etag","in",["etagValue1","etagValue2"]],["href","=","hrefValue"]`);
    });

    it("should add attribute prefix", () => {
        expect(new FilterBuilder()
            .addIn("etag", ["etagValue1", "etagValue2"], "myPrefix")
            .getFilters())
            .toEqual(`["myPrefix.etag","in",["etagValue1","etagValue2"]]`);
    });
});
