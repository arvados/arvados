// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Axios from "axios";
import { mockConfig } from "common/config";
import { createBrowserHistory } from "history";
import { dataExplorerActions } from "store/data-explorer/data-explorer-action";
import { GROUPS_PANEL_ID } from "./groups-panel-actions";
import { configureStore } from "store/store";
import { createServices } from "services/services";
import { getResource } from "store/resources/resources";

describe("GroupsPanelMiddlewareService", () => {
    let axiosInst;
    let store;
    let services;
    const config = {};
    const actions = {
        progressFn: (id, working) => { },
        errorFn: (id, message) => { }
    };

    beforeEach(() => {
        axiosInst = Axios.create({ headers: {} });
        services = createServices(mockConfig({}), actions, axiosInst);
        store = configureStore(createBrowserHistory(), services, config);
    });

    it("requests group member counts and updates resource store", async () => {
        // Given
        const fakeUuid = "zzzzz-j7d0g-000000000000000";
        axiosInst.get = cy.spy((url) => {
            if (url === '/groups') {
                return Promise.resolve(
                    { data: {
                        kind: "",
                        offset: 0,
                        limit: 100,
                        items: [{
                            can_manage: true,
                            can_write: true,
                            created_at: "2023-11-15T20:57:01.723043000Z",
                            delete_at: null,
                            description: null,
                            etag: "0000000000000000000000000",
                            frozen_by_uuid: null,
                            group_class: "role",
                            is_trashed: false,
                            kind: "arvados#group",
                            modified_at: "2023-11-15T20:57:01.719986000Z",
                            modified_by_user_uuid: "zzzzz-tpzed-000000000000000",
                            name: "Test Group",
                            owner_uuid: "zzzzz-tpzed-000000000000000",
                            properties: {},
                            trash_at: null,
                            uuid: fakeUuid,
                            writable_by: [
                                "zzzzz-tpzed-000000000000000",
                            ]
                        }],
                        items_available: 1,
                    }});
            } else if (url === '/links') {
                return Promise.resolve(
                    { data: {
                        items: [],
                        items_available: 234,
                        kind: "arvados#linkList",
                        limit: 0,
                        offset: 0
                    }});
            } else {
                return Promise.resolve(
                    { data: {}});
            }
        });

        // When
        await store.dispatch(dataExplorerActions.REQUEST_ITEMS({id: GROUPS_PANEL_ID}));
        // Wait for async fetching of group count promises to resolve
        await new Promise(setImmediate);

        // Expect
        expect(axiosInst.get).to.be.calledThrice;
        expect(axiosInst.get.getCall(0).args[0]).to.equal('/groups');
        expect(axiosInst.get.getCall(0).args[1].params).to.deep.include({count: 'none'});
        expect(axiosInst.get.getCall(1).args[0]).to.equal('/groups');
        expect(axiosInst.get.getCall(1).args[1].params).to.deep.include({count: 'exact', limit: 0});
        expect(axiosInst.get.getCall(2).args[0]).to.equal('/links');
        const group = getResource(fakeUuid)(store.getState().resources);
        expect(group?.memberCount).to.equal(234);
    });

    it('requests group member count and stores null on failure', async () => {
        // Given
        const fakeUuid = "zzzzz-j7d0g-000000000000000";
        axiosInst.get = cy.spy((url) => {
            if (url === '/groups') {
                return Promise.resolve(
                    { data: {
                        kind: "",
                        offset: 0,
                        limit: 100,
                        items: [{
                            can_manage: true,
                            can_write: true,
                            created_at: "2023-11-15T20:57:01.723043000Z",
                            delete_at: null,
                            description: null,
                            etag: "0000000000000000000000000",
                            frozen_by_uuid: null,
                            group_class: "role",
                            is_trashed: false,
                            kind: "arvados#group",
                            modified_at: "2023-11-15T20:57:01.719986000Z",
                            modified_by_user_uuid: "zzzzz-tpzed-000000000000000",
                            name: "Test Group",
                            owner_uuid: "zzzzz-tpzed-000000000000000",
                            properties: {},
                            trash_at: null,
                            uuid: fakeUuid,
                            writable_by: [
                                "zzzzz-tpzed-000000000000000",
                            ]
                        }],
                        items_available: 1,
                    }});
            } else if (url === '/links') {
                return Promise.reject();
            } else {
                return Promise.resolve({ data: {}});
            }
        });

        // When
        await store.dispatch(dataExplorerActions.REQUEST_ITEMS({id: GROUPS_PANEL_ID}));
        // Wait for async fetching of group count promises to resolve
        await new Promise(setImmediate);

        // Expect
        expect(axiosInst.get).to.be.calledThrice;
        expect(axiosInst.get.getCall(0).args[0]).to.equal('/groups');
        expect(axiosInst.get.getCall(0).args[1].params).to.deep.include({count: 'none'});
        expect(axiosInst.get.getCall(1).args[0]).to.equal('/groups');
        expect(axiosInst.get.getCall(1).args[1].params).to.deep.include({count: 'exact', limit: 0});
        expect(axiosInst.get.getCall(2).args[0]).to.equal('/links');
        const group = getResource(fakeUuid)(store.getState().resources);
        expect(group?.memberCount).to.equal(null);
    });
});
