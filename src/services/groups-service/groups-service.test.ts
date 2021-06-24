// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { GroupsService } from "./groups-service";
import { ApiActions } from "services/api/api-actions";

describe("GroupsService", () => {

    const axiosMock = new MockAdapter(axios);

    const actions: ApiActions = {
        progressFn: (id: string, working: boolean) => {},
        errorFn: (id: string, message: string) => {}
    };

    beforeEach(() => {
        axiosMock.reset();
    });

    it("#contents", async () => {
        axiosMock
            .onGet("/groups/1/contents")
            .reply(200, {
                kind: "kind",
                offset: 2,
                limit: 10,
                items: [{
                    modified_at: "now"
                }],
                items_available: 20
            });

        const groupsService = new GroupsService(axios, actions);
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
