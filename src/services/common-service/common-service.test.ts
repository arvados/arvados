// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import { ApiActions } from "~/services/api/api-actions";
import { CommonService } from "./common-service";

const actions: ApiActions = {
    progressFn: (id: string, working: boolean) => {},
    errorFn: (id: string, message: string) => {}
};

describe("CommonService", () => {
    const axiosInstance = axios.create();
    // const axiosMock = new MockAdapter(axiosInstance);
    const commonService = new CommonService(axiosInstance, "resource", actions);

    it("throws an exception when passing uuid as empty string to get()", () => {
        expect(() => commonService.get("")).toThrowError("UUID cannot be empty string");
    });

    it("throws an exception when passing uuid as empty string to update()", () => {
        expect(() => commonService.update("", {})).toThrowError("UUID cannot be empty string");
    });

    it("throws an exception when passing uuid as empty string to delete()", () => {
        expect(() => commonService.delete("")).toThrowError("UUID cannot be empty string");
    });
});