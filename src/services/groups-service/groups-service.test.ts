// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import MockAdapter from "axios-mock-adapter/types";
import GroupsService from "./groups-service";

describe("GroupsService", () => {

    const axiosMock = new MockAdapter(axios);

    beforeEach(() => {
        axiosMock.reset();
    });

    it("#contents", async () => {
        axiosMock
            .onGet("/groups/1/contents/")
            .reply(200, {
                kind: "kind",
                offset: 2,
                limit: 10,
                items: [{
                    modified_at: "now"
                }],
                items_available: 20
            });

        const groupsService = new GroupsService(axios);
        const resource = await groupsService.contents("1", { limit: 10, offset: 1 });
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
