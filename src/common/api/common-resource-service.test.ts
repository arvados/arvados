// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import CommonResourceService from "./common-resource-service";
import axios from "axios";
import MockAdapter from "axios-mock-adapter";

describe("CommonResourceService", () => {

    const axiosMock = new MockAdapter(axios);

    beforeEach(() => {
        axiosMock.reset();
    });

    it("#delete", async () => {
        axiosMock
            .onDelete("/resource/uuid")
            .reply(200, { deleted_at: "now" });

        const commonResourceService = new CommonResourceService(axios, "resource");
        const resource = await commonResourceService.delete("uuid");
        expect(resource).toEqual({ deletedAt: "now" });
    });

    it("#get", async () => {
        axiosMock
            .onGet("/resource/uuid")
            .reply(200, { modified_at: "now" });

        const commonResourceService = new CommonResourceService(axios, "resource");
        const resource = await commonResourceService.get("uuid");
        expect(resource).toEqual({ modifiedAt: "now" });
    });

    it("#list", async () => {
        axiosMock
            .onGet("/resource")
            .reply(200, {
                kind: "kind",
                offset: 2,
                limit: 10,
                items: [{
                    modified_at: "now"
                }],
                items_available: 20
            });

        const commonResourceService = new CommonResourceService(axios, "resource");
        const resource = await commonResourceService.list({ limit: 10, offset: 1 });
        expect(resource).toEqual({
            kind: "kind",
            offset: 2,
            limit: 10,
            items: [{
                modifiedAt: "now"
            }],
            itemsAvailable: 20
        });
    });
});
