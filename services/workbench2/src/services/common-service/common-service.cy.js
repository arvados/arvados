// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonService } from "./common-service";

const actions = {
    progressFn: (id, working) => {},
    errorFn: (id, message) => {}
};

describe("CommonService", () => {
    let commonService;

    beforeEach(() => {
        commonService = new CommonService({}, "resource", actions);
    });

    it("throws an exception when passing uuid as empty string to get()", () => {
        expect(() => commonService.get("")).to.throw("UUID cannot be empty string");
    });

    it("throws an exception when passing uuid as empty string to update()", () => {
        expect(() => commonService.update("", {})).to.throw("UUID cannot be empty string");
    });

    it("throws an exception when passing uuid as empty string to delete()", () => {
        expect(() => commonService.delete("")).to.throw("UUID cannot be empty string");
    });
});