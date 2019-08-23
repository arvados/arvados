// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { joinUrls } from "~/services/api/url-builder";

describe("UrlBuilder", () => {
    it("should join urls properly 1", () => {
        expect(joinUrls('http://localhost:3000', '/main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 2", () => {
        expect(joinUrls('http://localhost:3000/', '/main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 3", () => {
        expect(joinUrls('http://localhost:3000//', '/main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 4", () => {
        expect(joinUrls('http://localhost:3000', '//main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 5", () => {
        expect(joinUrls('http://localhost:3000///', 'main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 6", () => {
        expect(joinUrls('http://localhost:3000///', '//main')).toEqual('http://localhost:3000/main');
    });
    it("should join urls properly 7", () => {
        expect(joinUrls(undefined, '//main')).toEqual('/main');
    });
    it("should join urls properly 8", () => {
        expect(joinUrls(undefined, 'main')).toEqual('/main');
    });
    it("should join urls properly 9", () => {
        expect(joinUrls('http://localhost:3000///', undefined)).toEqual('http://localhost:3000');
    });
});
