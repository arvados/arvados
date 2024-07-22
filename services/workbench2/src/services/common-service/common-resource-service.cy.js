// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CommonResourceService } from "./common-resource-service";
import axios from "axios";
import MockAdapter from "axios-mock-adapter";

const actions = {
    progressFn: (id, working) => {},
    errorFn: (id, message) => {}
};

export const mockResourceService = (
    Service => {
        const axiosInstance = axios.create();
        const service = new Service(axiosInstance, actions);
        Object.keys(service).map(key => service[key] = cy.stub());
        return service;
    });

describe("CommonResourceService", () => {
    let axiosInstance;
    let axiosMock;

    beforeEach(() => {
        axiosInstance = axios.create();
        axiosMock = new MockAdapter(axiosInstance);
    });

    it("#create", async () => {
        axiosMock
            .onPost("/resources")
            .reply(200, { owner_uuid: "ownerUuidValue" });

        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        const resource = await commonResourceService.create({ ownerUuid: "ownerUuidValue" });
        expect(resource).to.deep.equal({ ownerUuid: "ownerUuidValue" });
    });

    it("#create maps request params to snake case", async () => {
        cy.stub(axiosInstance, "post").returns(Promise.resolve({data: {}}));
        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        await commonResourceService.create({ ownerUuid: "ownerUuidValue" });
    });

    it("#create ignores fields listed as readonly", async () => {
        cy.stub(axiosInstance, "post").returns(Promise.resolve({data: {}}));
        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        // UUID fields are read-only on all resources.
        await commonResourceService.create({ uuid: "this should be ignored", ownerUuid: "ownerUuidValue" });
        expect(axiosInstance.post).to.be.calledWith("/resources", {resource: {owner_uuid: "ownerUuidValue"}});
    });

    it("#update ignores fields listed as readonly", async () => {
        cy.stub(axiosInstance, "put").returns(Promise.resolve({data: {}}));
        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        // UUID fields are read-only on all resources.
        await commonResourceService.update('resource-uuid', { uuid: "this should be ignored", ownerUuid: "ownerUuidValue" });
        expect(axiosInstance.put).to.be.calledWith("/resources/resource-uuid", {resource:  {owner_uuid: "ownerUuidValue"}});
    });

    it("#delete", async () => {
        axiosMock
            .onDelete("/resources/uuid")
            .reply(200, { deleted_at: "now" });

        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        const resource = await commonResourceService.delete("uuid");
        expect(resource).to.deep.equal({ deletedAt: "now" });
    });

    it("#get", async () => {
        axiosMock
            .onGet("/resources/uuid")
            .reply(200, {
                modified_at: "now",
                properties: {
                    responsible_owner_uuid: "another_owner"
                }
            });

        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        const resource = await commonResourceService.get("uuid");
        // Only first level keys are mapped to camel case
        expect(resource).to.deep.equal({
            modifiedAt: "now",
            properties: {
                responsible_owner_uuid: "another_owner"
            }
        });
    });

    it("#list", async () => {
        axiosMock
            .onGet("/resources")
            .reply(200, {
                kind: "kind",
                offset: 2,
                limit: 10,
                items: [{
                    modified_at: "now",
                    properties: {
                        is_active: true
                    }
                }],
                items_available: 20
            });

        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        const resource = await commonResourceService.list({ limit: 10, offset: 1 });
        // First level keys are mapped to camel case inside "items" arrays
        expect(resource).to.deep.equal({
            kind: "kind",
            offset: 2,
            limit: 10,
            items: [{
                modifiedAt: "now",
                properties: {
                    is_active: true
                }
            }],
            itemsAvailable: 20
        });
    });

    it("#list using POST when query string is too big", async () => {
        axiosMock
            .onAny("/resources")
            .reply(200);
        const tooBig = 'x'.repeat(1500);
        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        await commonResourceService.list({ filters: tooBig });
        expect(axiosMock.history.get.length).to.equal(0);
        expect(axiosMock.history.post.length).to.equal(1);
        const postParams = new URLSearchParams(axiosMock.history.post[0].data);
        expect(postParams.get('filters')).to.equal(`[${tooBig}]`);
        expect(postParams.get('_method')).to.equal('GET');
    });

    it("#list using GET when query string is not too big", async () => {
        axiosMock
            .onAny("/resources")
            .reply(200);
        const notTooBig = 'x'.repeat(1480);
        const commonResourceService = new CommonResourceService(axiosInstance, "resources", actions);
        await commonResourceService.list({ filters: notTooBig });
        expect(axiosMock.history.post.length).to.equal(0);
        expect(axiosMock.history.get.length).to.equal(1);
        expect(axiosMock.history.get[0].params.filters).to.equal(`[${notTooBig}]`);
    });
});
