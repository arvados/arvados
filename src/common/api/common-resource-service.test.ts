// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "./common-resource-service";
import axios, { AxiosInstance } from "axios";
import MockAdapter from "axios-mock-adapter";
import { Resource } from "../../models/resource";

export const mockResourceService = <R extends Resource, C extends CommonResourceService<R>>(Service: new (client: AxiosInstance) => C) => {
    const axiosInstance = axios.create();
    const axiosMock = new MockAdapter(axiosInstance);
    const service = new Service(axiosInstance);
    Object.keys(service).map(key => service[key] = jest.fn());
    return service;
};

describe("CommonResourceService", () => {
    const axiosInstance = axios.create();
    const axiosMock = new MockAdapter(axiosInstance);

    beforeEach(() => {
        axiosMock.reset();
    });

    it("#create", async () => {
        axiosMock
            .onPost("/resource/")
            .reply(200, { owner_uuid: "ownerUuidValue" });

        const commonResourceService = new CommonResourceService(axiosInstance, "resource");
        const resource = await commonResourceService.create({ ownerUuid: "ownerUuidValue" });
        expect(resource).toEqual({ ownerUuid: "ownerUuidValue" });
    });

    it("#create maps request params to snake case", async () => {
        axiosInstance.post = jest.fn(() => Promise.resolve({data: {}}));
        const commonResourceService = new CommonResourceService(axiosInstance, "resource");
        await commonResourceService.create({ ownerUuid: "ownerUuidValue" });
        expect(axiosInstance.post).toHaveBeenCalledWith("/resource/", {owner_uuid: "ownerUuidValue"});
    });

    it("#delete", async () => {
        axiosMock
            .onDelete("/resource/uuid")
            .reply(200, { deleted_at: "now" });

        const commonResourceService = new CommonResourceService(axiosInstance, "resource");
        const resource = await commonResourceService.delete("uuid");
        expect(resource).toEqual({ deletedAt: "now" });
    });

    it("#get", async () => {
        axiosMock
            .onGet("/resource/uuid")
            .reply(200, { modified_at: "now" });

        const commonResourceService = new CommonResourceService(axiosInstance, "resource");
        const resource = await commonResourceService.get("uuid");
        expect(resource).toEqual({ modifiedAt: "now" });
    });

    it("#list", async () => {
        axiosMock
            .onGet("/resource/")
            .reply(200, {
                kind: "kind",
                offset: 2,
                limit: 10,
                items: [{
                    modified_at: "now"
                }],
                items_available: 20
            });

        const commonResourceService = new CommonResourceService(axiosInstance, "resource");
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
