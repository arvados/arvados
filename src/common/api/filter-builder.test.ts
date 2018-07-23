// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { FilterBuilder } from "./filter-builder";

describe("FilterBuilder", () => {

    let filters: FilterBuilder;

    beforeEach(() => {
        filters = FilterBuilder.create();
    });

    it("should add 'equal' rule", () => {
        expect(
            filters.addEqual("etag", "etagValue").serialize()
        ).toEqual(`[["etag","=","etagValue"]]`);
    });

    it("should add 'like' rule", () => {
        expect(
            filters.addLike("etag", "etagValue").serialize()
        ).toEqual(`[["etag","like","%etagValue%"]]`);
    });

    it("should add 'ilike' rule", () => {
        expect(
            filters.addILike("etag", "etagValue").serialize()
        ).toEqual(`[["etag","ilike","%etagValue%"]]`);
    });

    it("should add 'is_a' rule", () => {
        expect(
            filters.addIsA("etag", "etagValue").serialize()
        ).toEqual(`[["etag","is_a","etagValue"]]`);
    });

    it("should add 'is_a' rule for set", () => {
        expect(
            filters.addIsA("etag", ["etagValue1", "etagValue2"]).serialize()
        ).toEqual(`[["etag","is_a",["etagValue1","etagValue2"]]]`);
    });

    it("should add 'in' rule", () => {
        expect(
            filters.addIn("etag", "etagValue").serialize()
        ).toEqual(`[["etag","in","etagValue"]]`);
    });

    it("should add 'in' rule for set", () => {
        expect(
            filters.addIn("etag", ["etagValue1", "etagValue2"]).serialize()
        ).toEqual(`[["etag","in",["etagValue1","etagValue2"]]]`);
    });

    it("should add multiple rules", () => {
        expect(
            filters
                .addIn("etag", ["etagValue1", "etagValue2"])
                .addEqual("href", "hrefValue")
                .serialize()
        ).toEqual(`[["etag","in",["etagValue1","etagValue2"]],["href","=","hrefValue"]]`);
    });

    it("should concatenate multiple builders", () => {
        expect(
            filters
                .concat(FilterBuilder.create().addIn("etag", ["etagValue1", "etagValue2"]))
                .concat(FilterBuilder.create().addEqual("href", "hrefValue"))
                .serialize()
        ).toEqual(`[["etag","in",["etagValue1","etagValue2"]],["href","=","hrefValue"]]`);
    });

    it("should add attribute prefix", () => {
        expect(FilterBuilder
            .create("myPrefix")
            .addIn("etag", ["etagValue1", "etagValue2"])
            .serialize())
            .toEqual(`[["my_prefix.etag","in",["etagValue1","etagValue2"]]]`);
    });




});
