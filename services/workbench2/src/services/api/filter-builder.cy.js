// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { FilterBuilder } from "./filter-builder";

describe("FilterBuilder", () => {

    let filters;

    beforeEach(() => {
        filters = new FilterBuilder();
    });

    it("should add 'equal' rule (string)", () => {
        expect(
            filters.addEqual("etag", "etagValue").getFilters()
        ).to.equal(`["etag","=","etagValue"]`);
    });

    it("should add 'equal' rule (boolean)", () => {
        expect(
            filters.addEqual("is_trashed", true).getFilters()
        ).to.equal(`["is_trashed","=",true]`);
    });

    it("should add 'like' rule", () => {
        expect(
            filters.addLike("etag", "etagValue").getFilters()
        ).to.equal(`["etag","like","%etagValue%"]`);
    });

    it("should add 'ilike' rule", () => {
        expect(
            filters.addILike("etag", "etagValue").getFilters()
        ).to.equal(`["etag","ilike","%etagValue%"]`);
    });

    it("should add 'contains' rule", () => {
        expect(
            filters.addContains("properties.someProp", "someValue").getFilters()
        ).to.equal(`["properties.someProp","contains","someValue"]`);
    });

    it("should add 'is_a' rule", () => {
        expect(
            filters.addIsA("etag", "etagValue").getFilters()
        ).to.equal(`["etag","is_a","etagValue"]`);
    });

    it("should add 'is_a' rule for set", () => {
        expect(
            filters.addIsA("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).to.equal(`["etag","is_a",["etagValue1","etagValue2"]]`);
    });

    it("should add 'in' rule", () => {
        expect(
            filters.addIn("etag", "etagValue").getFilters()
        ).to.equal(`["etag","in","etagValue"]`);
    });

    it("should add 'in' rule for set", () => {
        expect(
            filters.addIn("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).to.equal(`["etag","in",["etagValue1","etagValue2"]]`);
    });

    it("should add 'not in' rule for set", () => {
        expect(
            filters.addNotIn("etag", ["etagValue1", "etagValue2"]).getFilters()
        ).to.equal(`["etag","not in",["etagValue1","etagValue2"]]`);
    });

    it("should add multiple rules", () => {
        expect(
            filters
                .addIn("etag", ["etagValue1", "etagValue2"])
                .addEqual("href", "hrefValue")
                .getFilters()
        ).to.equal(`["etag","in",["etagValue1","etagValue2"]],["href","=","hrefValue"]`);
    });

    it("should add attribute prefix", () => {
        expect(new FilterBuilder()
            .addIn("etag", ["etagValue1", "etagValue2"], "myPrefix")
            .getFilters())
            .to.equal(`["myPrefix.etag","in",["etagValue1","etagValue2"]]`);
    });

    it('should add full text search', () => {
        expect(
            new FilterBuilder()
                .addFullTextSearch('my custom search')
                .getFilters()
        ).to.equal(`["any","ilike","%my%"],["any","ilike","%custom%"],["any","ilike","%search%"]`);
    });
});
